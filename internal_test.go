package cli // Intentional

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
