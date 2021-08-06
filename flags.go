package cli

func names(name string) []string {
	return []string{name}
}

// flagName gets the long and short name for getopt given the name specified in the flag
func flagName(name string) (string, rune) {
	if len(name) == 1 {
		return "", []rune(name)[0]
	} else {
		return name, 0
	}
}
