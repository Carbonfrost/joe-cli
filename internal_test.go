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
	}, []string{}, true)
	c.initialize()
	return c
}

func InitializeCommand(f *Command) *Context {
	c := &Context{
		Context: context.TODO(),
	}
	c = c.copy(&commandContext{
		cmd: f,
	}, []string{}, true)
	c.initialize()
	return c
}
