package cli // Intentional

// Expose some members for testing

func ParseUsage(text string) *usage {
	return parseUsage(text)
}
