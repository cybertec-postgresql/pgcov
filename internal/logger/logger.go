package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Logger provides leveled logging functionality
type Logger struct {
	verbose bool
	info    *log.Logger
	debug   *log.Logger
	error   *log.Logger
}

var defaultLogger *Logger

func init() {
	defaultLogger = New(false, os.Stderr)
}

// New creates a new logger instance
func New(verbose bool, output io.Writer) *Logger {
	flags := log.Ldate | log.Ltime
	return &Logger{
		verbose: verbose,
		info:    log.New(output, "[INFO]  ", flags),
		debug:   log.New(output, "[DEBUG] ", flags),
		error:   log.New(output, "[ERROR] ", flags),
	}
}

// SetDefault sets the default logger instance
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Default returns the default logger instance
func Default() *Logger {
	return defaultLogger
}

// SetVerbose enables or disables verbose logging
func (l *Logger) SetVerbose(verbose bool) {
	l.verbose = verbose
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
}

// Info logs an informational message (always shown)
func (l *Logger) Info(format string, args ...interface{}) {
	l.info.Printf(format, args...)
}

// Debug logs a debug message (only shown if verbose is enabled)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		l.debug.Printf(format, args...)
	}
}

// Error logs an error message (always shown)
func (l *Logger) Error(format string, args ...interface{}) {
	l.error.Printf(format, args...)
}

// Debugf is an alias for Debug
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(format, args...)
}

// Infof is an alias for Info
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(format, args...)
}

// Errorf is an alias for Error
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(format, args...)
}

// Package-level functions that use the default logger

// SetVerbose enables or disables verbose logging on the default logger
func SetVerbose(verbose bool) {
	defaultLogger.SetVerbose(verbose)
}

// IsVerbose returns whether verbose logging is enabled on the default logger
func IsVerbose() bool {
	return defaultLogger.IsVerbose()
}

// Info logs an informational message using the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Debug logs a debug message using the default logger (only shown if verbose is enabled)
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Debugf is an alias for Debug
func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

// Infof is an alias for Info
func Infof(format string, args ...interface{}) {
	Info(format, args...)
}

// Errorf is an alias for Error
func Errorf(format string, args ...interface{}) {
	Error(format, args...)
}

// Printf is a general-purpose print function that respects verbose mode
func Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Println is a general-purpose print function that respects verbose mode
func Println(args ...interface{}) {
	fmt.Println(args...)
}
