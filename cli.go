package cli

import (
	"github.com/kballard/go-shellquote"
)

func Split(s string) ([]string, error) {
	return shellquote.Split(s)
}
