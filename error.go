// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"bytes"
	"cmp"
	"fmt"
	"strconv"
)

// ErrorCode provides common error codes in the CLI framework.
type ErrorCode int

// ParseError provides the common representation of errors during parsing
type ParseError struct {
	// Code is the code to use.
	Code ErrorCode

	// Err is the internal error, if any
	Err error

	// Name specifies the name of the flag, arg, command, or expression that
	// caused the error
	Name string

	// Value is the value that caused the error
	Value string

	// Remaining contains arguments which could not be parsed
	Remaining []string
}

// InternalError represents an error that has occurred because of the
// way the library is used rather than a user parse error. An example
// of this is ErrTimingTooLAte, which has occurred because an action
// was added to pipeline that wasn't acceptable.
type InternalError struct {
	// Path describes the path where the internal error occurred
	Path ContextPath

	// Timing specifies when the error is occurring
	Timing Timing

	// Err returns the internal error
	Err error
}

// ExitCoder is an error that knows how to convert to its exit code
type ExitCoder interface {
	error

	// ExitCode obtains the exit code for the error
	ExitCode() int
}

type exitError struct {
	message  any
	exitCode int
}

const (
	// UnexpectedArgument provides the error when an unexpected argument is encountered
	UnexpectedArgument = ErrorCode(iota)

	// CommandNotFound provides the error when the command is not found
	CommandNotFound

	// UnknownOption occurs when the option is not recognized
	UnknownOption

	// MissingArgument means that the value is required for a flag
	MissingArgument

	// InvalidArgument error represents the value for a position argument or flag not being parsable
	InvalidArgument

	// ExpectedArgument occurs when a value must be specified to a positional argument or flag
	ExpectedArgument

	// UnknownExpr represents an expression name that was not recognized
	UnknownExpr

	// ArgsMustPrecedeExprs occurs in expression parsing for unexpected arguments
	ArgsMustPrecedeExprs

	// FlagUsedAfterArgs occurs when a flag is used after a positional arg, but not allowed
	FlagUsedAfterArgs

	// ExpectedRequiredOption occurs when a flag or option is required to be specified
	ExpectedRequiredOption
)

// Exit formats an error message using the default formats for each of the arguments,
// except the last one, which is interpreted as the desired exit code.  The function
// provides similar semantics to fmt.Sprint in that all values are converted to text
// and joined together.  Spaces are added between operands when neither is a string.
// If the last argument is an integer, it is interpreted as the exit code that will
// be generated when the program exits.  If no integer is present, the value 1 is used.
func Exit(message ...any) ExitCoder {
	switch len(message) {
	case 0:
		return exitCore("", 1)
	case 1:
		switch msg := message[0].(type) {
		case ErrorCode:
			return &ParseError{
				Code: msg,
			}
		case ExitCoder:
			return msg
		case int:
			return exitCore("", msg)
		case nil:
			return exitCore("", 1)
		default:
			return exitCore(fmt.Sprint(msg), 1)
		}
	default:
		last := len(message) - 1
		switch code := message[last].(type) {
		case int:
			return exitCore(fmt.Sprint(message[0:last]...), code)
		case ErrorCode:
			return exitCore(fmt.Sprintf("%s: %s", code.formatError("", "", nil), fmt.Sprint(message[0:last]...)), 2)
		default:
			return exitCore(fmt.Sprint(message...), 1)
		}
	}
}

func exitCore(message string, code int) ExitCoder {
	if message == "" {
		message = fmt.Sprintf("exited with status %d", code)
	}
	return &exitError{
		message:  message,
		exitCode: code,
	}
}

// ExitCode always returns 2
func (e *ParseError) ExitCode() int {
	return 2
}

func (e *ParseError) Error() string {
	return e.Code.formatError(e.Name, e.Value, e.Err)
}

// Unwrap returns the internal error
func (e *ParseError) Unwrap() error {
	return e.Err
}

// formatError renders the user-facing error message for the code, weaving in the
// name and value of the offending flag, arg, command, or expression.  The cause,
// when present, is the inner error that explains the underlying failure (e.g. a
// value that could not be parsed by argTakerError, or the count required by
// expectedArgument); the message is composed here rather than delegated to it.
func (e ErrorCode) formatError(name, value string, cause error) string {
	switch e {
	case UnexpectedArgument:
		if value == "" {
			return "unexpected argument"
		}
		return fmt.Sprintf("unexpected argument %q", value)
	case ExpectedArgument:
		msg := "expected argument"
		if cause != nil {
			// TODO Would be ideal to not use cause for this (see expectedArgument, which carries the count)
			msg = cause.Error()
		}
		if name == "" {
			return msg
		}
		return fmt.Sprintf("%s for %s", msg, name)
	case CommandNotFound:
		if name == "" {
			return "not a command"
		}
		return fmt.Sprintf("%q is not a command", name)
	case UnknownOption:
		if name == "" {
			return "unknown option"
		}
		return fmt.Sprintf("unknown option: %s", name)
	case MissingArgument:
		return "missing parameter"
	case InvalidArgument:
		if cause != nil {
			return cause.Error()
		}
		if name == "" {
			return "parameter not valid"
		}
		return fmt.Sprintf("option %s does not take a value", name)
	case UnknownExpr:
		if name == "" {
			return "unknown expression"
		}
		return fmt.Sprintf("unknown expression: %s", name)
	case ArgsMustPrecedeExprs:
		if value == "" {
			return "arguments must precede expressions"
		}
		return fmt.Sprintf("arguments must precede expressions: %q", value)
	case FlagUsedAfterArgs:
		if name == "" {
			return "flag used after arguments"
		}
		return fmt.Sprintf("can't use %s after arguments", name)
	case ExpectedRequiredOption:
		if name == "" {
			return "required and must be specified"
		}
		return fmt.Sprintf("%s is required and must be specified", name)
	}
	return "unknown error"
}

func (e *exitError) Error() string {
	return fmt.Sprintf("%v", e.message)
}

func (e *exitError) ExitCode() int {
	return e.exitCode
}

func (i *InternalError) Unwrap() error {
	return i.Err
}

func (i *InternalError) Error() string {
	return fmt.Sprintf(
		"internal error, at %q (%v): %v",
		i.Path.String(),
		i.Timing.Describe(),
		i.Err)
}

func commandMissing(name string) error {
	return &ParseError{
		Code: CommandNotFound,
		Name: name,
	}
}

func unexpectedArgument(value string, remaining []string) *ParseError {
	return &ParseError{
		Code:      UnexpectedArgument,
		Remaining: remaining,
		Value:     value,
	}
}

func expectedRequiredOption(name string) *ParseError {
	return &ParseError{
		Code: ExpectedRequiredOption,
		Name: name,
	}
}

func flagUnexpectedArgument(name string, value string, remaining []string) *ParseError {
	return &ParseError{
		Code:      InvalidArgument,
		Remaining: remaining,
		Name:      name,
		Value:     value,
	}
}

func expectedArgument(count int) *ParseError {
	w := "argument"
	if count > 1 {
		w = fmt.Sprint(count, " arguments")
	}
	return &ParseError{
		Code: ExpectedArgument,
		Err:  fmt.Errorf("expected %s", w),
	}
}

func unknownOption(name any, remaining []string) error {
	nameStr := optionName(name)
	return &ParseError{
		Code:      UnknownOption,
		Name:      nameStr,
		Remaining: remaining,
	}
}

func flagAfterArgError(name any) error {
	nameStr := optionName(name)
	return &ParseError{
		Code: FlagUsedAfterArgs,
		Name: nameStr,
	}
}

func argTakerError(name string, value string, err error, remaining []string) error {
	if p, ok := err.(*ParseError); ok {
		p.Name = name
		p.Value = value
		p.Remaining = remaining
		return p
	}
	return &ParseError{
		Code:      InvalidArgument,
		Name:      name,
		Value:     value,
		Err:       err,
		Remaining: remaining,
	}
}

func formatStrconvError(err error, value string) error {
	if e, ok := err.(*strconv.NumError); ok {
		switch e.Err {
		case strconv.ErrRange:
			err = fmt.Errorf("value out of range: %s", value)
		case strconv.ErrSyntax:
			if value == "" {
				err = fmt.Errorf("empty string is not a valid number")
			} else {
				err = fmt.Errorf("not a valid number: %s", value)
			}
		}
	}
	return err
}

func optionName(name any) string {
	switch n := name.(type) {
	case rune:
		if n == '-' {
			return "-"
		}
		return "-" + string(n)
	case string:
		if len(n) == 1 {
			return "-" + string(n)
		}
		return "--" + n
	}
	panic("unreachable!")
}

func listOfValues(values []string, useQuotes bool, conjopt ...string) string {
	conj := cmp.Or(cmp.Or(conjopt...), "or")
	quotes := [2]string{"", ""}
	if useQuotes {
		quotes[0], quotes[1] = "`", "'"
	}
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	case 2:
		return fmt.Sprintf("%[1]s%[3]s%[2]s %[5]s %[1]s%[4]s%[2]s", quotes[0], quotes[1], values[0], values[1], conj)
	default:
		var b bytes.Buffer
		for i, v := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			if i == len(values)-1 {
				b.WriteString(conj)
				b.WriteString(" ")
			}
			b.WriteString(quotes[0])
			b.WriteString(v)
			b.WriteString(quotes[1])
		}
		return b.String()
	}
}

var _ error = (*InternalError)(nil)
