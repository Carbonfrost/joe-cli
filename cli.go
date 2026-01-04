// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	"encoding"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
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

// Quote uses shell escaping rules if necessary to quote a value
func Quote(v any) string {
	switch p := v.(type) {
	case nil:
		return ""
	case fmt.Stringer: // includes Value
		return p.String()
	case encoding.TextMarshaler:
		s, _ := p.MarshalText()
		return string(s)
	}

	v = dereference(v)

	switch s := v.(type) {
	case string:
		if s == "" {
			return "''"
		}
		if !unsafeShlexChars.Match([]byte(s)) {
			return s
		}

		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	case []string:
		return strings.Join(s, ",")
	case []byte:
		return hex.EncodeToString(s)
	case map[string]string:
		return support.FormatMap(s, ",")
	case []*NameValue:
		val := make([]string, len(s))
		for i, nvp := range s {
			val[i] = nvp.String()
		}
		return strings.Join(val, ",")

	case bool:
	case int, int8, int16, int32, int64:
	case uint, uint8, uint16, uint32, uint64:
	case float32, float64:
	case *url.URL:
	case net.IP:
	case *regexp.Regexp:
	case *big.Int, *big.Float:
		// These all support fmt.Sprint below
	}
	return fmt.Sprint(v)
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

// Shell detects the shell which is running.
func Shell() string {
	return support.DetermineShell()
}

var unsafeShlexChars = regexp.MustCompile(`[^\w@%+=:,./-]`)
var errorNotTty = Exit("stdin not tty")
