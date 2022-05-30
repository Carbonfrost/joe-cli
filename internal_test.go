package cli // Intentional

import (
	"context"
	"io/fs"
	"os"
)

// Expose some members for testing

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

func NewFlagSynopsis(long string) *flagSynopsis {
	return &flagSynopsis{
		Long:    long,
		Primary: long,
		Names:   []string{"--" + long},
		Value:   &valueSynopsis{},
	}
}

func InitializeFlag(f *Flag) *Context {
	c := &Context{
		Context: context.TODO(),
	}
	c = c.copy(&optionContext{
		option: f,
	}, true)
	c.initialize()
	return c
}

func InitializeCommand(f *Command) *Context {
	c := &Context{
		Context: context.TODO(),
	}
	c = c.copy(&commandContext{
		cmd: f,
	}, true)
	c.initialize()
	return c
}

// DefaultFS is the FS that is expected to be created when no
// other is set up
func DefaultFS() fs.FS {
	return newDefaultFS(os.Stdin, NewWriter(os.Stdout))
}
