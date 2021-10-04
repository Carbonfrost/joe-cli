package cli

import (
	"regexp"
	"strings"

	"github.com/kballard/go-shellquote"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Split splits the specified text using shell splitting rules
func Split(s string) ([]string, error) {
	return shellquote.Split(s)
}

// Join together the arguments, wrapping each in quotes if necessary
func Join(args []string) string {
	quoted := make([]string, len(args))
	for i, s := range args {
		quoted[i] = Quote(s)
	}
	return strings.Join(quoted, " ")
}

// Quote uses shell escaping rules if necessary to quote the string
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	if !unsafeShlexChars.Match([]byte(s)) {
		return s
	}

	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

var unsafeShlexChars = regexp.MustCompile(`[^\w@%+=:,./-]`)
