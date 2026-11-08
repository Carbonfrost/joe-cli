// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Format names the [log/slog] handler which backs a Logger.
type Format int

// The available formats for log output
const (
	// TextFormat writes logs using [slog.TextHandler].  This is the default.
	TextFormat Format = iota

	// JSONFormat writes logs using [slog.JSONHandler].
	JSONFormat

	maxFormat
)

var logFormatNames = [maxFormat]string{
	TextFormat: "text",
	JSONFormat: "json",
}

// NewHandler creates the handler which corresponds to the format
func (f Format) NewHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if f == JSONFormat {
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}

// String provides the name of the format
func (f Format) String() string {
	if f < 0 || f >= maxFormat {
		return ""
	}
	return logFormatNames[f]
}

// MarshalText provides the textual representation
func (f Format) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText converts the textual representation
func (f *Format) UnmarshalText(b []byte) error {
	name := strings.TrimSpace(string(b))
	for i, n := range logFormatNames {
		if n == name {
			*f = Format(i)
			return nil
		}
	}
	return fmt.Errorf("unexpected log format %q", name)
}
