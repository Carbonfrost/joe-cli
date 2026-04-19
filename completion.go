// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"context"
	"fmt"
	"iter"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

var (
	shellUnsafeChars = regexp.MustCompile(`\W*`)
)

// CompletionType enumerates the supported kinds of completions
type CompletionType int

// Completion is the shell auto-complete function for the flag, arg, or value
type Completion interface {
	Complete(context.Context) []CompletionItem
}

// StandardCompletion enumerates standard completion results
type StandardCompletion int

// CompletionRequest provides information about the completion request
type CompletionRequest struct {
	// Args that have been passed to the command so far
	Args []string

	// Incomplete the token that is incomplete being completed
	Incomplete string

	// Bindings gets the bindings that were partially parsed
	Bindings BindingMap

	// Err gets the error that occurred during parsing, likely *ParserError
	Err error
}

// CompletionFunc provide a function that can be used as a Completer
type CompletionFunc func(*Context) []CompletionItem

// Completion types
const (
	CompletionTypeToken CompletionType = iota
	CompletionTypeFile
	CompletionTypeDirectory
)

const (
	// FileCompletion generates a list of files in the completion response
	FileCompletion StandardCompletion = iota

	// DirectoryCompletion generations a list of directories in the completion response
	DirectoryCompletion
)

// CompletionItem defines an item displayed in the completion results
type CompletionItem struct {
	Type     CompletionType
	Value    string
	HelpText string

	// PreventSpaceAfter disables the addition of a space after the completion token
	PreventSpaceAfter bool
}

// ShellComplete provides the implementation of the shell-specific
// completion handler
type ShellComplete interface {
	GetCompletionRequest() (args []string, incomplete string)
	SetOptions(map[string]string)
	FormatCompletions([]CompletionItem) string
	GetSourceTemplate() *Template
}

type completionData struct {
	App struct {
		Name string
	}
	CompletionFunc   string
	JoeCompletionVar string
	Shell            string
	ShellComplete    ShellComplete
}

const (
	shellCompletesKey = "__ShellCompletes"
)

// CompletionRequest gets the completion request from the
// context if a completion is being requested.
func (c *Context) CompletionRequest() *CompletionRequest {
	return c.request
}

func (c *Context) clearCompletionRequest() {
	c.request = nil
}

func newCompletionData(c *Context) *completionData {
	appName := c.App().Name
	slug := strings.ReplaceAll(shellUnsafeChars.ReplaceAllString(appName, ""), "-", "_")
	envVar := fmt.Sprintf("_JOE_%s_COMPLETE", strings.ToUpper(slug))
	shell := os.Getenv(envVar)

	return &completionData{
		App: struct{ Name string }{
			Name: appName,
		},
		Shell:            shell,
		ShellComplete:    c.shellCompletes()[shell],
		CompletionFunc:   fmt.Sprintf("_%s_completion_func", slug),
		JoeCompletionVar: envVar,
	}
}

// ShellCompleteIntegration provides an action which renders the particular
// shell complete integration
func ShellCompleteIntegration(name string, s ShellComplete) Action {
	return Pipeline(
		func(c *Context) {
			c.shellCompletes()[name] = s
		},
		&Prototype{
			Name:    name + "-complete",
			Options: Hidden | Exits,
			Value:   new(bool),
		},
		At(ActionTiming, ActionFunc(func(c *Context) error {
			data := newCompletionData(c)
			tpl := s.GetSourceTemplate()
			return tpl.Execute(c.Stdout, data)
		})),
	)
}

// ApplyShellCompletion detects whether a dynamic shell completion request has been added to
// the environment and activates the corresponding supported response.
func ApplyShellCompletion() Action {
	return Setup{
		Uses: func(c *Context) {
			cc := newCompletionData(c)
			if cc.Shell != "" {
				setupRobustParsingMode(c)
			}
		},
	}
}

// SetCompletion sets completion for the command, flag, or arg.
func SetCompletion(c Completion) Action {
	return ActionFunc(func(ctx *Context) error {
		ctx.target.setCompletion(c)
		return nil
	})
}

func setupCompletion(c context.Context) error {
	return Do(c, Pipeline(
		optionalFlag("zsh-completion", ShellCompleteIntegration("zsh", newZshComplete())),
		ApplyShellCompletion(),
	))
}

func setupRobustParsingMode(c *Context) {
	// Activate robust parsing which causes errors in parsing to be
	// ignored so that we have an incomplete binding
	c.target.setInternalFlags(internalFlagRobustParseModeEnabled, true)
}

func completionToken(v any) CompletionItem {
	var helpText string
	if ht, ok := v.(interface{ HelpText() string }); ok {
		helpText = ht.HelpText()
	}
	return CompletionItem{
		Type:     CompletionTypeToken,
		HelpText: helpText,
		Value:    fmt.Sprint(v),
	}
}

// TokenGeneratorCompletion generates a list of tokens as completion results.
func TokenGeneratorCompletion[T any](fn func(*Context) []T) Completion {
	return CompletionFunc(func(c *Context) []CompletionItem {
		items := fn(c)
		return filterCompletionOnContext(
			c,
			func(yield func(CompletionItem) bool) {
				for _, s := range items {
					if !yield(completionToken(s)) {
						return
					}
				}
			})
	})
}

// ValueCompletion provides the context-specific completion values for
// the given strings.  This can be specified as the Completion for flags
// or args.  For flags, the name of the flag is automatically
// prefixed to the completion value using valid syntax.
func ValueCompletion[T any](values ...T) Completion {
	return CompletionFunc(func(c *Context) []CompletionItem {
		return filterCompletionOnContext(
			c,
			func(yield func(CompletionItem) bool) {
				for _, v := range values {
					if !yield(completionToken(v)) {
						return
					}
				}
			},
		)
	})
}

func filterCompletionOnContext(c *Context, values iter.Seq[CompletionItem]) []CompletionItem {
	incompletePart := c.CompletionRequest().Incomplete
	if incompletePart == "" {
		return slices.Collect(values)
	}

	if o, ok := c.Target().(*Flag); ok {
		res := make([]CompletionItem, 0)
		for _, n := range o.synopsis().Names {
			var prefix string
			if len(n) == 2 { // as in -s short names
				prefix = n // Force run-in style, which is most compatible
			} else {
				prefix = n + "="
			}
			for a := range values {
				v := prefix + a.Value
				if strings.HasPrefix(v, incompletePart) {
					item := a
					item.Value = v
					res = append(res, item)
				}
			}
		}
		return res
	}

	res := make([]CompletionItem, 0)
	for v := range values {
		if strings.HasPrefix(v.Value, incompletePart) {
			res = append(res, v)
		}
	}
	return res
}

// Complete considers the given arguments and completion request to determine
// completion items
func (c *Context) Complete(args []string, incomplete string) []CompletionItem {
	return c.complete(args, incomplete, c.parse(args))
}

func (c *Context) complete(args []string, incomplete string, re *robustParseResult) []CompletionItem {
	cc := &CompletionRequest{
		Args:       args,
		Incomplete: incomplete,
		Bindings:   re.bindings,
		Err:        re.err,
	}
	c.request = cc
	defer c.clearCompletionRequest()

	return c.target.completion().Complete(c)
}

func (c *Context) robustParsingMode() bool {
	return c.flagSetOrAncestor((internalFlags).robustParseModeEnabled)
}

func (c *Context) shellCompletes() map[string]ShellComplete {
	l, ok := c.LookupData(shellCompletesKey)
	if !ok {
		l = map[string]ShellComplete{}
		c.SetData(shellCompletesKey, l)
	}
	return l.(map[string]ShellComplete)
}

func (f CompletionFunc) Complete(c context.Context) []CompletionItem {
	if f == nil {
		return nil
	}
	return f(FromContext(c))
}

func (f CompletionFunc) Execute(ctx context.Context) error {
	return Do(ctx, SetCompletion(f))
}

func (s StandardCompletion) Complete(ctx context.Context) []CompletionItem {
	c := FromContext(ctx).CompletionRequest()
	switch s {
	case FileCompletion:
		return []CompletionItem{
			{Value: c.Incomplete, Type: CompletionTypeFile},
		}
	case DirectoryCompletion:
		return []CompletionItem{
			{Value: c.Incomplete, Type: CompletionTypeDirectory},
		}
	}
	panic(fmt.Sprintf("unexpected value: %v", s))
}

func (s StandardCompletion) Execute(ctx context.Context) error {
	return Do(ctx, SetCompletion(s))
}

func actualCompletion(c Completion) Completion {
	if c != nil {
		return c
	}
	return CompletionFunc(nil)
}

func newSourceTemplate(name string, text string) *Template {
	t := template.New(name)
	return &Template{
		Template: template.Must(
			t.Funcs(withExecute(builtinFuncs, t)).Parse(text),
		),
		Debug: debugTemplates(),
	}
}

var (
	_ Completion = (CompletionFunc)(nil)
	_ Action     = (CompletionFunc)(nil)
	_ Completion = (StandardCompletion)(0)
)
