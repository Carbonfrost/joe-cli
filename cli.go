package cli

import (
	"github.com/kballard/go-shellquote"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

func Split(s string) ([]string, error) {
	return shellquote.Split(s)
}
