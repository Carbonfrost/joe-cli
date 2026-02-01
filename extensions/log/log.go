// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package log provides access to [log/slog] structured logging
// with opinionated defaults and integration with flags.
package log

import (
	"context"
	"log/slog"
)

type Level = slog.Level

// Debug logs a message with the given arguments at the Debug level
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// DebugContext logs a message with the given arguments at the Debug level, using the specified context
func DebugContext(ctx context.Context, msg string, args ...any) {
	slog.DebugContext(ctx, msg, args...)
}

// Error logs a message with the given arguments at the Error level
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// ErrorContext logs a message with the given arguments at the Error level, using the specified context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}

// Info logs a message with the given arguments at the Info level
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// InfoContext logs a message with the given arguments at the Info level, using the specified context
func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

// Log logs a message with the given arguments at the specified level
func Log(ctx context.Context, level Level, msg string, args ...any) {
	slog.Log(ctx, level, msg, args...)
}

// LogAttrs logs a message with the given arguments at the specified level
func LogAttrs(ctx context.Context, level Level, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, level, msg, attrs...)
}

// Warn logs a message with the given arguments at the Warn level
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// WarnContext logs a message with the given arguments at the Warn level, using the specified context
func WarnContext(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, args...)
}

// With returns a derived logger with attributes already set
func With(args ...any) *slog.Logger {
	return slog.With(args...)
}
