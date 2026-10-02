// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prompt provides an extension which defines the behavior
// of interactively prompting for options, particularly when the
// options are required and interactive filling is used as fallback.

package prompt

import (
	"context"
)

//go:generate go tool counterfeiter -generate

//counterfeiter:generate -o internal/promptfakes . Prompter

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
