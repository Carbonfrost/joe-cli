package cli

import (
	"regexp"
	"strings"

	"github.com/kballard/go-shellquote"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

func Split(s string) ([]string, error) {
	return shellquote.Split(s)
}

func Join(args []string) string {
	quoted := make([]string, len(args))
	for i, s := range args {
		quoted[i] = Quote(s)
	}
	return strings.Join(quoted, " ")
}

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
