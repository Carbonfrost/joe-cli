// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

var defaultOpts = []Option{
	WithDefaultAction(),
}

// Logger provides a named logger, typically retrieved from the context.  A
// logger delegates to [log/slog], creating the handler on demand from the
// options which have been applied to it.
//
// The default logger is named with the empty string.  It is the logger that
// the log functions in this package delegate to and the only one which obtains
// flags from FlagsAndArgs.
type Logger struct {
	// Action specifies the action which defines the action to run when this value
	// is added to a pipeline. Typically, this is an initializer set via WithDefaultAction
	cli.Action

	mu             sync.Mutex
	name           string
	format         LogFormat
	handlerOptions slog.HandlerOptions
	w              io.Writer

	// logger memoizes the underlying logger.  It is invalidated whenever an
	// option is applied.
	logger *slog.Logger
}

// Option defines an option for initialization of a Logger.  An option is also
// an action, which applies it to the logger it names within the context.
type Option interface {
	cli.Action
	apply(*Logger)
}

type optionFunc struct {
	// logger names the logger that the option applies to when the option is
	// used as an action
	logger string
	fn     func(*Logger)
}

// New creates a new logger.
// By default, adding the Logger to the pipeline registers it with its name and
// in the context, which is required for most use cases, and, for the default
// logger, adds flags.
func New(opts ...Option) *Logger {
	l := new(Logger)
	l.Apply(defaultOpts...)
	l.Apply(opts...)
	return l
}

// FromContext retrieves the logger with the given name from the context.  The
// default logger has the empty string as its name.  This function panics if
// the logger is not present in the context.
func FromContext(ctx context.Context, name string) *Logger {
	res, err := tryFromContext(ctx, name)
	if err != nil {
		panic(err)
	}
	return res
}

func tryFromContext(ctx context.Context, name string) (*Logger, error) {
	if ctx != nil {
		if res, ok := ctx.Value(loggerKey(name)).(*Logger); ok {
			return res, nil
		}
		if s, ok := tryServices(ctx); ok {
			if res, ok := s.Lookup(name); ok {
				return res, nil
			}
		}
	}
	if name == "" {
		return nil, fmt.Errorf("expected default logger not present in context")
	}
	return nil, fmt.Errorf("expected logger %q not present in context", name)
}

// ContextValue provides an action that registers the given logger with its
// name and sets it into the context.
func ContextValue(l *Logger) cli.Action {
	return cli.Pipeline(
		cli.WithContext(func(ctx context.Context) context.Context {
			register(ctx, l)
			return context.WithValue(ctx, loggerKey(l.Name()), l)
		}),

		// The context services aren't necessarily ready within the Uses
		// pipeline, so registration is repeated once they are
		cli.Before(cli.ActionOf(func(ctx context.Context) error {
			register(ctx, l)
			return nil
		})),
	)
}

// Apply will apply the given options to the logger
func (l *Logger) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(l)
	}
}

// Pipeline retrieves the logger's action as a pipeline
func (l *Logger) Pipeline() cli.Action {
	return cli.Pipeline(l.Action)
}

// Name obtains the name of the logger.  The default logger has the empty
// string as its name.
func (l *Logger) Name() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.name
}

// Logger obtains the underlying logger, which is created on demand from the
// options which have been applied.  As a special case, the [slog] default is
// used when the receiver is nil.
func (l *Logger) Logger() *slog.Logger {
	if l == nil {
		return slog.Default()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logger == nil {
		opts := l.handlerOptions
		l.logger = slog.New(l.format.NewHandler(cmp.Or[io.Writer](l.w, os.Stderr), &opts))
	}
	return l.logger
}

// Debug logs a message with the given arguments at the Debug level
func (l *Logger) Debug(msg string, args ...any) {
	emit(context.Background(), l.Logger(), LevelDebug, msg, args)
}

// DebugContext logs a message with the given arguments at the Debug level, using the specified context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, l.Logger(), LevelDebug, msg, args)
}

// Error logs a message with the given arguments at the Error level
func (l *Logger) Error(msg string, args ...any) {
	emit(context.Background(), l.Logger(), LevelError, msg, args)
}

// ErrorContext logs a message with the given arguments at the Error level, using the specified context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, l.Logger(), LevelError, msg, args)
}

// Info logs a message with the given arguments at the Info level
func (l *Logger) Info(msg string, args ...any) {
	emit(context.Background(), l.Logger(), LevelInfo, msg, args)
}

// InfoContext logs a message with the given arguments at the Info level, using the specified context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, l.Logger(), LevelInfo, msg, args)
}

// Log logs a message with the given arguments at the specified level
func (l *Logger) Log(ctx context.Context, level Level, msg string, args ...any) {
	emit(ctx, l.Logger(), level, msg, args)
}

// LogAttrs logs a message with the given attributes at the specified level
func (l *Logger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...slog.Attr) {
	emitAttrs(ctx, l.Logger(), level, msg, attrs)
}

// Warn logs a message with the given arguments at the Warn level
func (l *Logger) Warn(msg string, args ...any) {
	emit(context.Background(), l.Logger(), LevelWarn, msg, args)
}

// WarnContext logs a message with the given arguments at the Warn level, using the specified context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	emit(ctx, l.Logger(), LevelWarn, msg, args)
}

// With returns a derived logger with attributes already set
func (l *Logger) With(args ...any) *slog.Logger {
	return l.Logger().With(args...)
}

func (f *optionFunc) apply(l *Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f.fn(l)

	// Invalidate the memoized logger so that the option takes effect
	l.logger = nil
}

func (f *optionFunc) Execute(c context.Context) error {
	f.apply(FromContext(c, f.logger))
	return nil
}

// WithName names the logger.  The default logger has the empty string as its
// name.  This option is meant for use with New; renaming a logger which has
// already been registered doesn't change its registration.
func WithName(name string) Option {
	return &optionFunc{fn: func(l *Logger) {
		l.name = name
	}}
}

// WithAddSource sets whether the handler computes the source file and line of
// the log statement, corresponding to [slog.HandlerOptions].AddSource.
func WithAddSource(v bool) Option {
	return withAddSource("", v)
}

// WithLevel sets the minimum level of the messages which are logged,
// corresponding to [slog.HandlerOptions].Level.
func WithLevel(v Level) Option {
	return withLevel("", v)
}

// WithLogFormat sets the format of the log output, which determines the
// handler that the logger uses.
func WithLogFormat(v LogFormat) Option {
	return withLogFormat("", v)
}

// WithOutput sets the writer that the logger writes to.  By default, this is
// the Stderr of the app.
func WithOutput(w io.Writer) Option {
	return &optionFunc{fn: func(l *Logger) {
		l.w = w
	}}
}

// WithAction sets the Action to use
func WithAction(v cli.Action) Option {
	return &optionFunc{fn: func(l *Logger) {
		l.Action = v
	}}
}

// WithDefaultAction sets the default action, which registers the logger with
// its name and in the context, and, if it is the default logger, also adds the
// flags from FlagsAndArgs.
func WithDefaultAction() Option {
	return &optionFunc{fn: func(l *Logger) {
		l.Action = cli.Pipeline(
			ContextValue(l),
			defaultOutput(l),
			defaultLoggerFlags(l),
		)
	}}
}

func withAddSource(logger string, v bool) *optionFunc {
	return &optionFunc{logger: logger, fn: func(l *Logger) {
		l.handlerOptions.AddSource = v
	}}
}

func withLevel(logger string, v Level) *optionFunc {
	return &optionFunc{logger: logger, fn: func(l *Logger) {
		l.handlerOptions.Level = v
	}}
}

func withLogFormat(logger string, v LogFormat) *optionFunc {
	return &optionFunc{logger: logger, fn: func(l *Logger) {
		l.format = v
	}}
}

func defaultOutput(l *Logger) cli.Action {
	return cli.ActionOf(func(ctx context.Context) error {
		c, ok := cli.TryFromContext(ctx)
		if !ok {
			return nil
		}

		l.mu.Lock()
		defer l.mu.Unlock()

		if l.w == nil {
			l.w = c.Stderr
			l.logger = nil
		}
		return nil
	})
}

// defaultLoggerFlags adds the flags, but only for the default logger.  The
// name isn't known until the options have been applied, which is why this is
// deferred to the time that the action runs.
func defaultLoggerFlags(l *Logger) cli.Action {
	return cli.ActionOf(func(ctx context.Context) error {
		if l.Name() != "" {
			return nil
		}
		return cli.Do(ctx, FlagsAndArgs())
	})
}

// FlagsAndArgs is an action which provides the default flags for the default
// logger to the application.  Despite its name, which is conventional, this
// action provides no args.
func FlagsAndArgs() cli.Action {
	return cli.AddFlags([]*cli.Flag{
		{Uses: SetLevel("")},
		{Uses: SetLogFormat("")},
		{Uses: SetAddSource("")},
	}...)
}

// SetLevel sets the minimum level of the logger named by name and provides
// reasonable defaults for initializing a flag.
func SetLevel(name string, v ...Level) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:       flagName(name, "level"),
			HelpText:   "Log messages at or above the specified LEVEL",
			Completion: cli.ValueCompletion("debug", "info", "warn", "error"),
		},
		withBinding(name, withLevel, v...),
	)
}

// SetLogFormat sets the format of the logger named by name and provides
// reasonable defaults for initializing a flag.
func SetLogFormat(name string, v ...LogFormat) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:       flagName(name, "format"),
			HelpText:   "Write log messages using the specified FORMAT",
			Completion: cli.ValueCompletion(logFormatNames[:]...),
		},
		withBinding(name, withLogFormat, v...),
	)
}

// SetAddSource causes the logger named by name to include the source position
// of log statements and provides reasonable defaults for initializing a flag.
func SetAddSource(name string, v ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     flagName(name, "add-source"),
			HelpText: "Include the source file and line in log output",
		},
		withBinding(name, withAddSource, v...),
	)
}

func withBinding[T any](name string, fn func(string, T) *optionFunc, v ...T) cli.Action {
	return bind.Action(func(value T) *optionFunc {
		return fn(name, value)
	}, bind.Exact(v...))
}

func flagName(logger, suffix string) string {
	return cmp.Or(logger, "log") + "-" + suffix
}
