package cli // Intentional

import (
	"context"
	"io/fs"
	"os"
)

// Expose some members for testing

// Provides the logic of os.Exit for tests
func SetOSExit(fn func(int)) {
	osExit = fn
}

func ParseUsage(text string) *usage {
	return parseUsage(text)
}

func SetInitialTiming(c *Context) {
	c.timing = InitialTiming
}

func SetBeforeTiming(c *Context) {
	c.timing = BeforeTiming
}

func SetAfterTiming(c *Context) {
	c.timing = AfterTiming
}

func SetActionTiming(c *Context) {
	c.timing = ActionTiming
}

func (a *Arg) ActualArgCounter() ArgCounter {
	return a.option.actualArgCounter()
}

func DefaultFlagCounter() ArgCounter {
	return &defaultCounter{requireSeen: true}
}

func NewFlagSynopsis(long string) *flagSynopsis {
	return &flagSynopsis{
		Long:    long,
		Primary: long,
		Names:   []string{"--" + long},
		Value:   &valueSynopsis{},
	}
}

func InitializeFlag(f *Flag) *Context {
	var captured *Context
	app := &App{
		Flags: []*Flag{
			f,
		},
	}
	f.Use(ActionFunc(func(c *Context) error {
		captured = c
		return nil
	}))
	app.Initialize(context.Background())
	return captured
}

func InitializeCommand(f *Command) *Context {
	var captured *Context
	app := &App{
		Commands: []*Command{
			f,
		},
	}
	f.Use(ActionFunc(func(c *Context) error {
		captured = c
		return nil
	}))
	app.Initialize(context.Background())
	return captured
}

// DefaultFS is the FS that is expected to be created when no
// other is set up
func DefaultFS() fs.FS {
	return newDefaultFS(os.Stdin, NewWriter(os.Stdout))
}
