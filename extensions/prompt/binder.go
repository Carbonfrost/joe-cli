// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompt

import (
	"context"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

// ActionBinder is a binder which displays a prompt using the [Prompter] in the
// context (see [ContextValue]).  There is a function that provides an
// ActionBinder corresponding to each method of Prompter.
//
// As a binder, a prompt is displayed when the value is bound at whichever
// timing you want, and its result is the value.
//
// As an action, its behavior depends upon the timing:
//
//   - In the Uses or Before pipeline, the prompt is scheduled to actually
//     display at the Action timing.
//   - Only when directly used within the ImplicitValueTiming (e.g. with
//     [cli.Implicitly]) is the prompt displayed before the action of a command
//     or other flags. The main use case is to fallback on prompts for missing
//     user inputs.  If no prompter is in the context, nothing happens and
//     no error is returned.
//   - In the Action or After timings, the prompt is displayed immediately; however,
//     if no prompter is in the context, an error is returned.
//
// When the prompt is displayed for a flag or arg, its value is replaced with
// the result of the prompt, or the error from the prompter is returned.  For a
// command, nothing additional happens, which is valid but unusual outside of
// a binding.
type ActionBinder[T any] = bind.ActionBinder[T]

type promptBinder[T, V any] struct {
	prompt func(context.Context, Prompter) (T, error)
	value  func(T) V
}

// Input provides an action binder for [Prompter.Input].
func Input(prompt, defaultValue string) ActionBinder[string] {
	return newPromptBinder(func(c context.Context, p Prompter) (string, error) {
		return p.Input(c, prompt, defaultValue)
	}, identity)
}

// Password provides an action binder for [Prompter.Password].
func Password(prompt string) ActionBinder[string] {
	return newPromptBinder(func(c context.Context, p Prompter) (string, error) {
		return p.Password(c, prompt)
	}, identity)
}

// Confirm provides an action binder for [Prompter.Confirm].
func Confirm(prompt string, defaultValue bool) ActionBinder[bool] {
	return newPromptBinder(func(c context.Context, p Prompter) (bool, error) {
		return p.Confirm(c, prompt, defaultValue)
	}, identity)
}

// SelectBinder is the action binder for [Prompter.Select].  As a binder,
// it provides the index of the selected option.
type SelectBinder struct {
	binder  ActionBinder[int]
	options []string
}

// MultiSelectBinder is the action binder for [Prompter.MultiSelect].  As a
// binder, it provides the indexes of the selected options.
type MultiSelectBinder struct {
	binder  ActionBinder[[]int]
	options []string
}

// Select provides an action binder for [Prompter.Select].  The value
// of a flag or arg is set to the selected option.
func Select(prompt, defaultValue string, options []string) *SelectBinder {
	return &SelectBinder{
		binder: newPromptBinder(func(c context.Context, p Prompter) (int, error) {
			return p.Select(c, prompt, defaultValue, options)
		}, func(i int) string {
			return options[i]
		}),
		options: options,
	}
}

// MultiSelect provides an action binder for [Prompter.MultiSelect].  The value
// of a flag or arg is set to the selected options.
func MultiSelect(prompt string, defaults, options []string) *MultiSelectBinder {
	return &MultiSelectBinder{
		binder: newPromptBinder(func(c context.Context, p Prompter) ([]int, error) {
			return p.MultiSelect(c, prompt, defaults, options)
		}, func(indexes []int) []string {
			return selectedOptions(options, indexes)
		}),
		options: options,
	}
}

// Edit provides an action binder for [Prompter.Edit].
func Edit(prompt, defaultValue string, blankAllowed bool) ActionBinder[string] {
	return newPromptBinder(func(c context.Context, p Prompter) (string, error) {
		return p.Edit(c, prompt, defaultValue, blankAllowed)
	}, identity)
}

func newPromptBinder[T, V any](prompt func(context.Context, Prompter) (T, error), value func(T) V) ActionBinder[T] {
	return &promptBinder[T, V]{
		prompt: prompt,
		value:  value,
	}
}

func (b *promptBinder[T, _]) Bind(ctx context.Context) (T, error) {
	p, err := tryFromContext(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	return b.prompt(ctx, p)
}

func (b *promptBinder[_, V]) Execute(ctx context.Context) error {
	c := cli.FromContext(ctx)
	switch {
	case c.IsImplicitTiming():
		if _, err := tryFromContext(c); err != nil {
			return nil
		}
		return b.display(c)

	case c.IsInitializing():
		if !c.IsCommand() {
			if err := c.Do(cli.Prototype{Value: new(V)}); err != nil {
				return err
			}
		}
		return c.At(cli.ActionTiming, cli.ActionFunc(b.display))

	case c.IsBefore():
		return c.At(cli.ActionTiming, cli.ActionFunc(b.display))

	default:
		return b.display(c)
	}
}

func (b *promptBinder[_, _]) display(c *cli.Context) error {
	res, err := b.Bind(c)
	if err != nil {
		return err
	}
	if c.IsCommand() {
		return nil
	}
	if !c.IsImplicitTiming() {
		// Replace rather than merge with the value from the command line
		if err := c.SetValue(nil); err != nil {
			return err
		}
	}
	return c.SetValue(b.value(res))
}

// Bind displays the prompt and obtains the index of the selected option
func (b *SelectBinder) Bind(ctx context.Context) (int, error) {
	return b.binder.Bind(ctx)
}

// Execute provides the behavior of the action (see [ActionBinder])
func (b *SelectBinder) Execute(ctx context.Context) error {
	return b.binder.Execute(ctx)
}

// Index provides a binder which obtains the index of the selected option
func (b *SelectBinder) Index() bind.Binder[int] {
	return bind.Seq(b.binder, func(i int) (int, error) {
		return i, nil
	})
}

// Value provides a binder which obtains the selected option
func (b *SelectBinder) Value() bind.Binder[string] {
	return bind.Seq(b.binder, func(i int) (string, error) {
		return b.options[i], nil
	})
}

// Bind displays the prompt and obtains the indexes of the selected options
func (b *MultiSelectBinder) Bind(ctx context.Context) ([]int, error) {
	return b.binder.Bind(ctx)
}

// Execute provides the behavior of the action (see [ActionBinder])
func (b *MultiSelectBinder) Execute(ctx context.Context) error {
	return b.binder.Execute(ctx)
}

// Index provides a binder which obtains the indexes of the selected options
func (b *MultiSelectBinder) Indexes() bind.Binder[[]int] {
	return bind.Seq(b.binder, func(indexes []int) ([]int, error) {
		return indexes, nil
	})
}

// Value provides a binder which obtains the selected options
func (b *MultiSelectBinder) Values() bind.Binder[[]string] {
	return bind.Seq(b.binder, func(indexes []int) ([]string, error) {
		return selectedOptions(b.options, indexes), nil
	})
}

func selectedOptions(options []string, indexes []int) []string {
	res := make([]string, len(indexes))
	for i, index := range indexes {
		res[i] = options[index]
	}
	return res
}

func identity[T any](v T) T {
	return v
}

var (
	_ ActionBinder[string] = (*promptBinder[string, string])(nil)
	_ ActionBinder[int]    = (*SelectBinder)(nil)
	_ ActionBinder[[]int]  = (*MultiSelectBinder)(nil)
)
