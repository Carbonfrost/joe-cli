// Package cli is a framework for command line applications.  joe-cli is designed to be
// easy to use and to extend.  It features a declarative model for organizing the app and
// a robust middleware/hook system to customize the app with reusable logic.
//
// This is the minimal, useful application:
//
//	func main() {
//	  app := &cli.App{
//	          Name: "greet",
//	          Action: func(c *cli.Context) error {
//	              fmt.Println("Hello, world!")
//	              return nil
//	          },
//	      }
//
//	  app.Run(os.Args)
//	}
package cli

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli/internal/support"
	"github.com/kballard/go-shellquote"
	"golang.org/x/term"
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

// SplitList considers escape sequences when splitting.  sep must not
// be empty string
func SplitList(s, sep string, n int) []string {
	return support.SplitList(s, sep, n)
}

// SplitMap considers escape sequences when splitting
func SplitMap(s string) map[string]string {
	i := support.SplitList(s, ",", -1)
	return support.ParseMap(i)
}

// ReadPasswordString securely gets a password, without the trailing '\n'.
// An error will be returned if the reader is not stdin connected to TTY.
func ReadPasswordString(in io.Reader) (string, error) {
	if f, ok := in.(interface{ Fd() uintptr }); ok {
		fd := int(f.Fd())
		if fd == 0 {
			data, err := term.ReadPassword(fd)
			return string(data), err
		}
	}
	return "", errorNotTty
}

// ReadString gets a line of text, without the trailing '\n'.
// An error will be returned if the reader is not stdin connected to TTY.
func ReadString(in io.Reader) (string, error) {
	if f, ok := in.(interface{ Fd() uintptr }); ok {
		fd := int(f.Fd())
		if fd == 0 {
			reader := bufio.NewReader(in)
			s, err := reader.ReadString('\n')
			if err != nil {
				return "", err
			}
			return s[0 : len(s)-1], nil
		}
	}

	return "", errorNotTty
}

var unsafeShlexChars = regexp.MustCompile(`[^\w@%+=:,./-]`)
var errorNotTty = Exit("stdin not tty")
