package cli // Intentional

import (
	"context"
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

func NewFlagSynopsis(long string) *flagSynopsis {
	return &flagSynopsis{
		long:  long,
		value: &valueSynopsis{},
	}
}

func InitializeFlag(f *Flag) *Context {
	c := &Context{
		Context: context.TODO(),
		internal: &flagContext{
			option: f,
			args_:  []string{},
		},
	}
	c.initialize()
	return c
}
