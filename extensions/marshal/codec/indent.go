// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import (
	"fmt"
	"strings"
)

// IndentStyle identifies the whitespace character used for indentation.
type IndentStyle int

// The available indent styles
const (
	// IndentSpace indents using space characters. This is the default.
	IndentSpace IndentStyle = iota

	// IndentTab indents using tab characters.
	IndentTab
)

func (s IndentStyle) unit() string {
	if s == IndentTab {
		return "\t"
	}
	return " "
}

// String returns the name of the indent style.
func (s IndentStyle) String() string {
	if s == IndentTab {
		return "tab"
	}
	return "space"
}

// MarshalText provides the textual representation
func (s IndentStyle) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText converts the textual representation. In addition to the
// canonical values "space" and "tab", the common misspellings "spaces" and
// "tabs" are also accepted.
func (s *IndentStyle) UnmarshalText(b []byte) error {
	switch strings.TrimSpace(string(b)) {
	case "space", "spaces":
		*s = IndentSpace
	case "tab", "tabs":
		*s = IndentTab
	default:
		return fmt.Errorf("unexpected indent style %q", string(b))
	}
	return nil
}
