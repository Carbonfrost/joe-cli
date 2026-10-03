// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prompt provides an extension which defines the behavior
// of interactively prompting for options, particularly when the
// options are required and interactive filling is used as fallback.

package prompt

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
)

type key string

const (
	contextPrompterKey key = "contextPrompter"
)

//go:generate go tool counterfeiter -generate

//counterfeiter:generate -o ../../internal/promptfakes . Prompter

// Prompter displays a prompt for user input.
type Prompter interface {
	// Input displays a prompt to the user for text
	Input(ctx context.Context, prompt, defaultValue string) (string, error)
	// Password displays a prompt to the user to securely enter a password
	Password(ctx context.Context, prompt string) (string, error)
	// Confirm displays a confirmation prompt
	Confirm(ctx context.Context, prompt string, defaultValue bool) (bool, error)
	// Select displays a prompt with a list of value options. At least
	// one option must be specified, and the defaultValue must be
	// among the options.
	Select(ctx context.Context, prompt, defaultValue string, options []string) (int, error)

	// MultiSelect displays a prompt with a list of value options.
	// At least one option must be specified, and the
	// values in the defaults must be among the options.
	MultiSelect(ctx context.Context, prompt string, defaults, options []string) ([]int, error)

	// Edit displays the system text editor to capture input.
	Edit(ctx context.Context, prompt, defaultValue string, blankAllowed bool) (string, error)
}

// ContextFilter provides the context filters in this package
type ContextFilter int

const (
	// Defines provides a context filter for flags defined by this package
	Defines ContextFilter = iota

	// Confirmed provides a context filter which detects whether the command
	// or any of its ancestors has been confirmed, typically by using the
	// flag defined by SetConfirmed
	Confirmed
)

var (
	pkgPath = reflect.TypeFor[ContextFilter]().PkgPath()
	tagged  = cli.Data(SourceAnnotation())

	definesImpl   = cli.HasData(SourceAnnotation())
	confirmedImpl = cli.HasData(confirmedDataKey, true)

	contextFilterLabels = map[ContextFilter][2]string{
		Defines:   {"defined in joe-cli/prompt pkg", "prompt.DEFINES"},
		Confirmed: {"confirmed", "prompt.CONFIRMED"},
	}
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

// Matches the context filter
func (f ContextFilter) Matches(ctx context.Context) bool {
	switch f {
	case Defines:
		return definesImpl.Matches(ctx)
	case Confirmed:
		return confirmedImpl.Matches(ctx)
	}
	return false
}

// String produces a textual representation of the context filter
func (f ContextFilter) String() string {
	return contextFilterLabels[f][1]
}

// MarshalText provides the textual representation
func (f ContextFilter) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText converts the textual representation
func (f *ContextFilter) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, v := range contextFilterLabels {
		if v[1] == token {
			*f = k
			return nil
		}
	}
	return nil
}

// Describe produces a representation of the context filter suitable for use in messages
func (f ContextFilter) Describe() string {
	return contextFilterLabels[f][0]
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is Prompter.
func ContextValue(v Prompter) cli.Action {
	return cli.WithContextValue(contextPrompterKey, v)
}

// FromContext retrieves the prompter from the context.  This panics if the
// prompter has not been registered, which is done by using [ContextValue].
func FromContext(ctx context.Context) Prompter {
	res, err := tryFromContext(ctx)
	if err != nil {
		panic(err)
	}
	return res
}

func tryFromContext(ctx context.Context) (Prompter, error) {
	if res, ok := ctx.Value(contextPrompterKey).(Prompter); ok {
		return res, nil
	}
	return nil, fmt.Errorf("expected %s value not present in context", contextPrompterKey)
}

var (
	_ encoding.TextMarshaler   = (*ContextFilter)(nil)
	_ encoding.TextUnmarshaler = (*ContextFilter)(nil)
	_ cli.ContextFilter        = (*ContextFilter)(nil)
)
