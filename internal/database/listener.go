package database

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cybertec-postgresql/pgcov/pkg/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Listener handles PostgreSQL LISTEN/NOTIFY for coverage signals
type Listener struct {
	conn           *pgx.Conn
	channel        string
	signals        chan types.CoverageSignal
	errors         chan error
	cancel         context.CancelFunc
	droppedSignals atomic.Int64
}

// NewListener creates a new LISTEN/NOTIFY listener using the config from a pool.
func NewListener(ctx context.Context, pool *pgxpool.Pool, channel string) (*Listener, error) {
	connCfg := pool.Config().ConnConfig.Copy()

	listener := &Listener{
		channel: channel,
		signals: make(chan types.CoverageSignal, 1000), // Buffered to avoid blocking
		errors:  make(chan error, 10),
	}

	// Register the OnNotification callback before connecting so that
	// every notification is dispatched to our channel automatically.
	connCfg.OnNotification = listener.handleNotification

	conn, err := pgx.ConnectConfig(ctx, connCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect for LISTEN: %w", err)
	}
	listener.conn = conn

	// Start listening on channel
	_, err = conn.Exec(ctx, fmt.Sprintf("LISTEN %s", channel))
	if err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to execute LISTEN: %w", err)
	}

	// Derive a cancellable context so Close can interrupt the receive loop.
	loopCtx, cancel := context.WithCancel(ctx)
	listener.cancel = cancel

	// Start background goroutine to drive reads (required for OnNotification to fire).
	go listener.receiveLoop(loopCtx)

	return listener, nil
}

// handleNotification is the pgx OnNotification callback.  It is invoked
// synchronously during WaitForNotification whenever a NOTIFY arrives.
func (l *Listener) handleNotification(_ *pgconn.PgConn, n *pgconn.Notification) {
	if n.Channel != l.channel {
		return
	}

	signal := types.CoverageSignal{
		SignalID:  n.Payload,
		Timestamp: time.Now(),
	}

	select {
	case l.signals <- signal:
	default:
		// Buffer full — increment counter so the caller can
		// detect and report lost signals after test execution.
		l.droppedSignals.Add(1)
	}
}

// receiveLoop blocks on WaitForNotification so that the pgx connection
// continuously reads from the server and dispatches OnNotification callbacks.
func (l *Listener) receiveLoop(ctx context.Context) {
	defer close(l.signals)
	defer close(l.errors)

	for {
		_, err := l.conn.WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil || l.conn.IsClosed() {
				return
			}

			select {
			case l.errors <- fmt.Errorf("notification error: %w", err):
			default:
			}
		}
	}
}

// Signals returns a channel that receives coverage signals
func (l *Listener) Signals() <-chan types.CoverageSignal {
	return l.signals
}

// Errors returns a channel that receives listener errors
func (l *Listener) Errors() <-chan error {
	return l.errors
}

// DroppedSignals returns the number of signals that were dropped because
// the internal buffer was full.  A non-zero value means coverage data is
// incomplete.
func (l *Listener) DroppedSignals() int64 {
	return l.droppedSignals.Load()
}

// Close stops the listener and closes the connection
func (l *Listener) Close(ctx context.Context) error {
	l.cancel() // interrupt receiveLoop's WaitForNotification

	// Unlisten
	if l.conn != nil && !l.conn.IsClosed() {
		_, _ = l.conn.Exec(ctx, fmt.Sprintf("UNLISTEN %s", l.channel))
		return l.conn.Close(ctx)
	}

	return nil
}

// WaitForSignal waits for a specific signal with timeout
func (l *Listener) WaitForSignal(ctx context.Context, timeout time.Duration) (*types.CoverageSignal, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case signal := <-l.signals:
		return &signal, nil
	case err := <-l.errors:
		return nil, err
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for signal")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CollectSignals collects all signals until context is cancelled or timeout
func (l *Listener) CollectSignals(ctx context.Context, timeout time.Duration) ([]types.CoverageSignal, error) {
	var signals []types.CoverageSignal

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case signal, ok := <-l.signals:
			if !ok {
				return signals, nil
			}
			signals = append(signals, signal)
		case err := <-l.errors:
			// Log error but continue collecting
			_ = err
		case <-timer.C:
			return signals, nil
		case <-ctx.Done():
			return signals, ctx.Err()
		}
	}
}

// Ping verifies the listener connection is alive
func (l *Listener) Ping(ctx context.Context) error {
	if l.conn == nil || l.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
	}
	return l.conn.Ping(ctx)
}

// SendTestNotification sends a test notification (for debugging)
func SendTestNotification(ctx context.Context, conn *pgconn.PgConn, channel string, payload string) error {
	sql := fmt.Sprintf("NOTIFY %s, '%s'", channel, payload)
	_, err := conn.Exec(ctx, sql).ReadAll()
	return err
}
