// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package log provides access to [log/slog] structured logging
// with opinionated defaults and integration with flags.
//
// The [Logger] is a context service, which works like the codec provider in
// the marshal extension: it has a default action, accumulates [Option] values
// which initialize it, and is added to and retrieved from the context.  Unlike
// that provider, loggers are named, which allows an app to provide more than
// one.  The default logger has the empty string as its name and is retrieved
// with FromContext(ctx, "").
//
// The log functions in this package delegate to the default logger.  Those
// which take a context obtain it from the context; those which don't bridge to
// the current app in order to find it.  When there is no default logger, they
// fall back to the [slog] default.
package log

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
)

// Level is the severity of a log message
type Level = slog.Level

// The names of the commonly used levels
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Debug logs a message with the given arguments at the Debug level
func Debug(msg string, args ...any) {
	emit(context.Background(), currentLogger(), LevelDebug, msg, args)
}

// DebugContext logs a message with the given arguments at the Debug level, using the specified context
func DebugContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, contextLogger(ctx), LevelDebug, msg, args)
}

// Error logs a message with the given arguments at the Error level
func Error(msg string, args ...any) {
	emit(context.Background(), currentLogger(), LevelError, msg, args)
}

// ErrorContext logs a message with the given arguments at the Error level, using the specified context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, contextLogger(ctx), LevelError, msg, args)
}

// Info logs a message with the given arguments at the Info level
func Info(msg string, args ...any) {
	emit(context.Background(), currentLogger(), LevelInfo, msg, args)
}

// InfoContext logs a message with the given arguments at the Info level, using the specified context
func InfoContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, contextLogger(ctx), LevelInfo, msg, args)
}

// Log logs a message with the given arguments at the specified level
func Log(ctx context.Context, level Level, msg string, args ...any) {
	emit(ctx, contextLogger(ctx), level, msg, args)
}

// LogAttrs logs a message with the given arguments at the specified level
func LogAttrs(ctx context.Context, level Level, msg string, attrs ...slog.Attr) {
	emitAttrs(ctx, contextLogger(ctx), level, msg, attrs)
}

// Warn logs a message with the given arguments at the Warn level
func Warn(msg string, args ...any) {
	emit(context.Background(), currentLogger(), LevelWarn, msg, args)
}

// WarnContext logs a message with the given arguments at the Warn level, using the specified context
func WarnContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, contextLogger(ctx), LevelWarn, msg, args)
}

// With returns a derived logger with attributes already set
func With(args ...any) *slog.Logger {
	return currentLogger().With(args...)
}

// emit writes the record to the logger.  It must be called directly from the
// function which the caller invoked so that the source position which is
// recorded belongs to that caller rather than to this package.
func emit(ctx context.Context, logger *slog.Logger, level Level, msg string, args []any) {
	if !logger.Enabled(ctx, level) {
		return
	}
	r := newRecord(level, msg)
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}

func emitAttrs(ctx context.Context, logger *slog.Logger, level Level, msg string, attrs []slog.Attr) {
	if !logger.Enabled(ctx, level) {
		return
	}
	r := newRecord(level, msg)
	r.AddAttrs(attrs...)
	_ = logger.Handler().Handle(ctx, r)
}

func newRecord(level Level, msg string) slog.Record {
	// Skip runtime.Callers, newRecord, emit, and the log function itself
	var pcs [1]uintptr
	runtime.Callers(4, pcs[:])

	return slog.NewRecord(time.Now(), level, msg, pcs[0])
}

// Action returns an action that wraps log.Log for convenient use in pipelines
func Action(level Level, msg string, args ...any) cli.Action {
	return cli.ActionOf(func(c context.Context) error {
		Log(c, level, msg, args...)
		return nil
	})
}
