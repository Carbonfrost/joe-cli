// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package printer provides the expression operators which print the values
// which flow through an expression evaluation pipeline.  These are the
// familiar operators from the Unix find command:
//
//	-fprint FILE
//	-fprint0 FILE
//	-fprintf FILE PATTERN
//	-print
//	-print0
//	-printf PATTERN
//
// Each of these operators always yields the value it was given, which makes it
// possible to compose them with the rest of the pipeline.
//
// The [Printer] does the actual work.  It is a context service, which means
// that it is registered into the context when it is added to a pipeline and
// retrieved from the context with [FromContext].  Adding it to a pipeline also
// registers the expression operators listed above.  A minimal app resembles:
//
//	app := &cli.App{
//	    Uses: printer.New(),
//	    Args: []*cli.Arg{
//	        {
//	            Name: "path",
//	            NArg: 1,
//	        },
//	        {
//	            Name:  "expression",
//	            Value: new(expr.Expression),
//	        },
//	    },
//	    Action: func(c *cli.Context) error {
//	        return expr.FromContext(c, "expression").Evaluate(c, time.Now())
//	    },
//	}
//
// Invoking this app as
//
//	app . -fprintf test.txt %(year)/%(month)
//
// writes the year and month of the value which was pushed into the pipeline to
// the file test.txt.  How the value is converted into the expander which
// resolves the names within a pattern is controlled by
// [WithExpanderFactory]; how patterns themselves are parsed is controlled
// by [WithPatternOptions].
package printer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"sync"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

type key string

const (
	contextPrinterKey key = "contextPrinter"

	// nul terminates values printed by the -print0 and -fprint0 operators
	nul = "\x00"

	newline = "\n"
)

var (
	pkgPath = reflect.TypeFor[Printer]().PkgPath()
	tagged  = cli.Data(SourceAnnotation())
)

var (
	// DefaultExpanderFactory provides the default conversion of a value from the
	// expression evaluation pipeline into the expander which expands patterns.  A
	// value which is itself an expander is used as-is; any other value uses the
	// reflection expander adapter [expander.Reflect], which resolves keys to the
	// fields and no-arg methods of the value.
	DefaultExpanderFactory ExpanderFactory = defaultExpanderFactory
)

// ExpanderFactory converts a value from the expression evaluation pipeline
// into the expander which is used to expand patterns.
type ExpanderFactory func(context.Context, any) expander.Interface

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all expression operators that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

// Printer facilitates the fprint-style expression operators.  It resolves the
// expander to use for each value in the expression evaluation pipeline, it
// compiles patterns, and it owns the files which the operators name.  Use
// [New] to create one which registers itself as a context service and
// registers the expression operators.
//
// A file is created the first time that it is named and it is held open for
// the remainder of the run, which causes each subsequent print to append to
// it.  Use [Printer.Close] to close the files, which the default action does
// automatically in the After timing.
type Printer struct {
	// Action provides the action to run when the printer is added to a
	// pipeline.
	cli.Action

	patternOpts []expander.Option
	factory     ExpanderFactory
	fs          cli.FS

	mu       sync.Mutex
	writers  map[string]io.Writer
	open     []fs.File
	patterns map[string]*expander.Pattern
}

// Option provides an option for configuring the printer.
type Option func(*Printer)

// New creates a printer which uses the default action.  The default action
// registers the printer as a context service, adds each of the expression
// operators via [AddExprs], and closes any files which were opened when the
// app exits.
func New(opts ...Option) *Printer {
	p := &Printer{}
	p.Apply(defaultOptions()...)
	p.Apply(opts...)
	return p
}

// Apply will apply the given options to the printer.
func (p *Printer) Apply(opts ...Option) {
	for _, o := range opts {
		o(p)
	}
}

// Pipeline obtains the pipeline which sets up the printer.
func (p *Printer) Pipeline() cli.Action {
	return p.Action
}

func defaultOptions() []Option {
	return []Option{
		WithDefaultAction(),
	}
}

// WithAction sets the action to use with the printer.
func WithAction(a cli.Action) Option {
	return func(p *Printer) {
		p.Action = cli.ActionOf(a) // allow action to be nil
	}
}

// WithDefaultAction sets the action to the default, which sets the printer
// into the context, adds the expression operators, and closes any files which
// the printer opened.
func WithDefaultAction() Option {
	return func(p *Printer) {
		p.Action = cli.Pipeline(
			ContextValue(p),
			AddExprs(),
			cli.After(cli.ActionOf(p.Close)),
		)
	}
}

// WithPatternOptions sets the options which are used when the patterns named
// by the expression operators are parsed.  Refer to [expander.Compile] for the
// available options.  This option is additive; each call appends to the
// options which are already present.
func WithPatternOptions(opts ...expander.Option) Option {
	return func(p *Printer) {
		p.patternOpts = append(p.patternOpts, opts...)
	}
}

// WithExpanderFactory sets how a value from the expression evaluation pipeline
// is converted into the expander which is used to expand patterns.  When
// unset, [DefaultExpanderFactory] is used.
func WithExpanderFactory(fn func(context.Context, any) expander.Interface) Option {
	return func(p *Printer) {
		p.factory = ExpanderFactory(fn)
	}
}

// WithFS sets the file system which is used to interpret the file names given
// to the expression operators.  When unset, the file system from the cli
// context is used, which also provides the convention that the file named with
// a dash refers to standard output.
func WithFS(f cli.FS) Option {
	return func(p *Printer) {
		p.fs = f
	}
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is *Printer.
func ContextValue(v *Printer) cli.Action {
	return cli.WithContextValue(contextPrinterKey, v)
}

// FromContext retrieves the printer from the context.  This panics if the
// printer has not been registered, which is done by adding it to a pipeline
// (see [New]) or by using [ContextValue].
func FromContext(ctx context.Context) *Printer {
	res, err := tryFromContext(ctx)
	if err != nil {
		panic(err)
	}
	return res
}

func tryFromContext(ctx context.Context) (*Printer, error) {
	if res, ok := ctx.Value(contextPrinterKey).(*Printer); ok {
		return res, nil
	}
	return nil, fmt.Errorf("expected %s value not present in context", contextPrinterKey)
}

// AddExprs provides an action which registers each of the expression operators
// which the printer supports: -fprint, -fprint0, -fprintf, -print, -print0,
// and -printf.
//
// When this action is used within the Uses pipeline of the arg which defines
// the expression, the operators are added to it directly.  Otherwise it adds
// a hook which will add it to any arg that has an Expression, which is what
// enables [New] to be useable at the level of the command or app.
func AddExprs() cli.Action {
	return cli.ActionOf(func(c *cli.Context) error {
		if definesExpression(c) {
			return cli.Do(c, addExprs())
		}
		return c.Customize("<>", cli.IfMatch(
			cli.ContextFilterFunc(definesExpression),
			addExprs(),
		))
	})
}

// Print provides an evaluator which prints the value from the expression
// evaluation pipeline to standard output followed by a newline.
func Print() expr.Evaluator {
	return evaluator(func(ctx context.Context, p *Printer, v any) error {
		return p.Print(ctx, v)
	})
}

// Print0 provides an evaluator which prints the value from the expression
// evaluation pipeline to standard output followed by a null character.
func Print0() expr.Evaluator {
	return evaluator(func(ctx context.Context, p *Printer, v any) error {
		return p.Print0(ctx, v)
	})
}

// Printf provides an evaluator which prints the value from the expression
// evaluation pipeline to standard output using the given pattern.
func Printf(pattern *expander.Pattern) expr.Evaluator {
	return evaluator(func(ctx context.Context, p *Printer, v any) error {
		return p.Printf(ctx, pattern, v)
	})
}

// Fprint provides an evaluator which prints the value from the expression
// evaluation pipeline to the named file followed by a newline.
func Fprint(file string) expr.Evaluator {
	return evaluator(func(ctx context.Context, p *Printer, v any) error {
		return p.Fprint(ctx, file, v)
	})
}

// Fprint0 provides an evaluator which prints the value from the expression
// evaluation pipeline to the named file followed by a null character.
func Fprint0(file string) expr.Evaluator {
	return evaluator(func(ctx context.Context, p *Printer, v any) error {
		return p.Fprint0(ctx, file, v)
	})
}

// Fprintf provides an evaluator which prints the value from the expression
// evaluation pipeline to the named file using the given pattern.
func Fprintf(file string, pattern *expander.Pattern) expr.Evaluator {
	return evaluator(func(ctx context.Context, p *Printer, v any) error {
		return p.Fprintf(ctx, file, pattern, v)
	})
}

// Print prints the value to standard output followed by a newline.
func (p *Printer) Print(ctx context.Context, v any) error {
	return printValue(p.stdout(ctx), v, newline)
}

// Print0 prints the value to standard output followed by a null character.
func (p *Printer) Print0(ctx context.Context, v any) error {
	return printValue(p.stdout(ctx), v, nul)
}

// Printf prints the value to standard output after expanding the pattern with
// the expander which corresponds to the value.  The writer which is used
// understands the stdout and stderr control expressions (see
// [expander.Renderer]).
func (p *Printer) Printf(ctx context.Context, pattern *expander.Pattern, v any) error {
	stdout, stderr := p.stdio(ctx)
	_, err := pattern.Fprint(expander.NewRenderer(stdout, stderr), p.Expander(ctx, v))
	return err
}

// Fprint prints the value to the named file followed by a newline.
func (p *Printer) Fprint(ctx context.Context, file string, v any) error {
	w, err := p.file(ctx, file)
	if err != nil {
		return err
	}
	return printValue(w, v, newline)
}

// Fprint0 prints the value to the named file followed by a null character.
func (p *Printer) Fprint0(ctx context.Context, file string, v any) error {
	w, err := p.file(ctx, file)
	if err != nil {
		return err
	}
	return printValue(w, v, nul)
}

// Fprintf prints the value to the named file after expanding the pattern with
// the expander which corresponds to the value.
func (p *Printer) Fprintf(ctx context.Context, file string, pattern *expander.Pattern, v any) error {
	w, err := p.file(ctx, file)
	if err != nil {
		return err
	}
	_, err = pattern.Fprint(w, p.Expander(ctx, v))
	return err
}

// Expander obtains the expander for the given value from the expression
// evaluation pipeline using the expander factory.
func (p *Printer) Expander(ctx context.Context, v any) expander.Interface {
	if p.factory == nil {
		return DefaultExpanderFactory(ctx, v)
	}
	return p.factory(ctx, v)
}

// Compile compiles the given pattern using the pattern options which were
// configured for the printer.  Patterns are cached, so compiling the same
// pattern text obtains the same pattern.
func (p *Printer) Compile(pattern string) *expander.Pattern {
	p.mu.Lock()
	defer p.mu.Unlock()

	if res, ok := p.patterns[pattern]; ok {
		return res
	}
	res := expander.Compile(pattern, p.patternOpts...)
	if p.patterns == nil {
		p.patterns = map[string]*expander.Pattern{}
	}
	p.patterns[pattern] = res
	return res
}

// Close closes each of the files which the printer opened and forgets them,
// which means that naming one of these files again re-creates it.
func (p *Printer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for _, f := range p.open {
		errs = append(errs, f.Close())
	}
	p.open = nil
	p.writers = nil
	return errors.Join(errs...)
}

func (p *Printer) file(ctx context.Context, name string) (io.Writer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Open the file on first use or else retrieve it from cache
	if w, ok := p.writers[name]; ok {
		return w, nil
	}

	f, err := p.fileSystem(ctx).OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}

	w, ok := f.(io.Writer)
	if !ok {
		f.Close()
		return nil, fmt.Errorf("file not writable: %s", name)
	}

	if p.writers == nil {
		p.writers = map[string]io.Writer{}
	}
	p.writers[name] = w
	p.open = append(p.open, f)
	return w, nil
}

func (p *Printer) fileSystem(ctx context.Context) cli.FS {
	if p.fs != nil {
		return p.fs
	}
	if c, ok := cli.TryFromContext(ctx); ok {
		if actual := cli.NewFS(c.FS); actual != nil {
			return actual
		}
	}
	return cli.DirFS(".")
}

func (p *Printer) stdout(ctx context.Context) io.Writer {
	stdout, _ := p.stdio(ctx)
	return stdout
}

func (p *Printer) stdio(ctx context.Context) (stdout io.Writer, stderr io.Writer) {
	if c, ok := cli.TryFromContext(ctx); ok {
		return c.Stdout, c.Stderr
	}
	return os.Stdout, os.Stderr
}

func addExprs() cli.Action {
	// A thunk is used so that each expression which is added is a distinct
	// instance, which matters because expression operators hold on to the
	// state of their bindings
	return cli.ActionOf(func(c context.Context) error {
		return cli.Do(c, expr.AddExprs(newExprs()...))
	})
}

func newExprs() []*expr.Expr {
	return []*expr.Expr{
		{
			Name:     "fprint",
			HelpText: "Print the value to FILE",
			Args:     []*cli.Arg{{Name: "file"}},
			Evaluate: expr.BindEvaluator(Fprint, bind.String("file")),
			Uses:     tagged,
		},
		{
			Name:     "fprint0",
			HelpText: "Print the value to FILE terminated by a null character",
			Args:     []*cli.Arg{{Name: "file"}},
			Evaluate: expr.BindEvaluator(Fprint0, bind.String("file")),
			Uses:     tagged,
		},
		{
			Name:     "fprintf",
			HelpText: "Print the value to FILE using the format PATTERN",
			Args:     []*cli.Arg{{Name: "file"}, {Name: "pattern"}},
			Evaluate: expr.BindEvaluator2(Fprintf, bind.String("file"), bindPattern("pattern")),
			Uses:     tagged,
		},
		{
			Name:     "print",
			HelpText: "Print the value to standard output",
			Evaluate: Print(),
			Uses:     tagged,
		},
		{
			Name:     "print0",
			HelpText: "Print the value to standard output terminated by a null character",
			Evaluate: Print0(),
			Uses:     tagged,
		},
		{
			Name:     "printf",
			HelpText: "Print the value to standard output using the format PATTERN",
			Args:     []*cli.Arg{{Name: "pattern"}},
			Evaluate: expr.BindEvaluator(Printf, bindPattern("pattern")),
			Uses:     tagged,
		},
	}
}

func bindPattern(name any) bind.Binder[*expander.Pattern] {
	return bind.SeqContext(bind.String(name), func(ctx context.Context, text string) (*expander.Pattern, error) {
		p, err := tryFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return p.Compile(text), nil
	})
}

func evaluator(fn func(context.Context, *Printer, any) error) expr.Evaluator {
	return expr.EvaluatorOf(func(ctx context.Context, v any) error {
		p, err := tryFromContext(ctx)
		if err != nil {
			return err
		}
		return fn(ctx, p, v)
	})
}

func definesExpression(c *cli.Context) bool {
	a := c.Arg()
	if a == nil {
		return false
	}
	_, ok := a.Value.(*expr.Expression)
	return ok
}

func printValue(w io.Writer, v any, terminator string) error {
	_, err := fmt.Fprintf(w, "%v%s", v, terminator)
	return err
}

func defaultExpanderFactory(_ context.Context, v any) expander.Interface {
	if e, ok := v.(expander.Interface); ok {
		return e
	}
	return expander.Reflect(v)
}
