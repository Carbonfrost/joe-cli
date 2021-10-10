package cli // Intentional

import (
	"context"
)

// Expose some members for testing

func ParseUsage(text string) *usage {
	return parseUsage(text)
}

func SetInitialTiming(c *Context) {
	c.timing = initialTiming
}

func SetBeforeTiming(c *Context) {
	c.timing = beforeTiming
}

func SetAfterTiming(c *Context) {
	c.timing = afterTiming
}

func SetActionTiming(c *Context) {
	c.timing = actionTiming
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
