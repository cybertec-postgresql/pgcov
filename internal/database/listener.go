package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgcov/pkg/types"
)

// Listener handles PostgreSQL LISTEN/NOTIFY for coverage signals
type Listener struct {
	conn       *pgx.Conn
	channel    string
	signals    chan types.CoverageSignal
	errors     chan error
	done       chan struct{}
	connString string
}

// NewListener creates a new LISTEN/NOTIFY listener
func NewListener(ctx context.Context, connString string, channel string) (*Listener, error) {
	// Parse connection string
	config, err := pgx.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Connect to database
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect for LISTEN: %w", err)
	}

	// Start listening on channel
	_, err = conn.Exec(ctx, fmt.Sprintf("LISTEN %s", channel))
	if err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to execute LISTEN: %w", err)
	}

	listener := &Listener{
		conn:       conn,
		channel:    channel,
		signals:    make(chan types.CoverageSignal, 1000), // Buffered to avoid blocking
		errors:     make(chan error, 10),
		done:       make(chan struct{}),
		connString: connString,
	}

	// Start background goroutine to receive notifications
	go listener.receiveLoop(ctx)

	return listener, nil
}

// receiveLoop continuously receives notifications from PostgreSQL
func (l *Listener) receiveLoop(ctx context.Context) {
	defer close(l.signals)
	defer close(l.errors)

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		default:
			// Wait for notification with short timeout to allow checking done/ctx
			waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			notification, err := l.conn.WaitForNotification(waitCtx)
			cancel()

			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return
				}

				// Check if connection is closed
				if l.conn.IsClosed() {
					select {
					case l.errors <- fmt.Errorf("connection closed"):
					default:
					}
					return
				}

				// Timeout is expected, just continue
				if waitCtx.Err() == context.DeadlineExceeded {
					continue
				}

				// Send error but continue
				select {
				case l.errors <- fmt.Errorf("notification error: %w", err):
				default:
				}
				continue
			}

			if notification != nil && notification.Channel == l.channel {
				// Create coverage signal
				signal := types.CoverageSignal{
					SignalID:  notification.Payload,
					Timestamp: time.Now(),
				}

				// Send signal (non-blocking)
				select {
				case l.signals <- signal:
				default:
					// Buffer full, log warning but don't block
					select {
					case l.errors <- fmt.Errorf("signal buffer full, dropping signal: %s", notification.Payload):
					default:
					}
				}
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

// Close stops the listener and closes the connection
func (l *Listener) Close(ctx context.Context) error {
	close(l.done)

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
