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

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	slog.DebugContext(ctx, msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}
func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

func Log(ctx context.Context, level Level, msg string, args ...any) {
	slog.Log(ctx, level, msg, args...)
}

func LogAttrs(ctx context.Context, level Level, msg string, attrs ...slog.Attr) {
	slog.LogAttrs(ctx, level, msg, attrs...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, args...)
}

func With(args ...any) *slog.Logger {
	return slog.With(args...)
}
