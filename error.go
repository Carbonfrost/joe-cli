// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cli

import (
	"bytes"
	"errors"
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

type formattableError interface {
	error
	FillTemplate(*ParseError) error
}

type errorTemplate struct {
	format   string
	fallback string
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
			return exitCore(fmt.Sprintf("%s: %s", code.String(), fmt.Sprint(message[0:last]...)), 2)
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
	if e.Err == nil {
		return e.Code.String()
	}

	if t, ok := e.Err.(formattableError); ok {
		return t.FillTemplate(e).Error()
	}

	return e.Err.Error()
}

// Unwrap returns the internal error
func (e *ParseError) Unwrap() error {
	return e.Err
}

// String produces a textual representation of error code
func (e ErrorCode) String() string {
	switch e {
	case UnexpectedArgument:
		return "unexpected argument"
	case ExpectedArgument:
		return "expected argument"
	case CommandNotFound:
		return "is not a command"
	case UnknownOption:
		return "unknown option"
	case MissingArgument:
		return "missing parameter"
	case InvalidArgument:
		return "parameter not valid"
	case UnknownExpr:
		return "unknown expression"
	case ArgsMustPrecedeExprs:
		return "arguments must precede expressions"
	case FlagUsedAfterArgs:
		return "flag used after arguments"
	case ExpectedRequiredOption:
		return "is required and must be specified"
	}
	return "unknown error"
}

func (e *exitError) Error() string {
	return fmt.Sprintf("%v", e.message)
}

func (e *exitError) ExitCode() int {
	return e.exitCode
}

func (f errorTemplate) Error() string {
	return f.fallback
}

func (f errorTemplate) FillTemplate(p *ParseError) error {
	if p.Name == "" {
		return errors.New(f.fallback)
	}
	return fmt.Errorf(f.format, p.Name, p.Value)
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
		Err:  fmt.Errorf("%q is not a command", name),
		Name: name,
	}
}

func unexpectedArgument(value string, remaining []string) *ParseError {
	return &ParseError{
		Code:      UnexpectedArgument,
		Err:       fmt.Errorf("unexpected argument %q", value),
		Remaining: remaining,
		Value:     value,
	}
}

func expectedRequiredOption(name string) *ParseError {
	return &ParseError{
		Code: ExpectedRequiredOption,
		Err:  fmt.Errorf("%s %s", name, ExpectedRequiredOption),
	}
}

func flagUnexpectedArgument(name string, value string, remaining []string) *ParseError {
	return &ParseError{
		Code:      InvalidArgument,
		Err:       fmt.Errorf("option %s does not take a value", name),
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
	fallback := fmt.Sprintf("expected %s", w)
	return &ParseError{
		Code: ExpectedArgument,
		Err: errorTemplate{
			fallback: fallback,
			format:   fallback + " for %[1]s",
		},
	}
}

func unknownOption(name any, remaining []string) error {
	nameStr := optionName(name)
	return &ParseError{
		Code:      UnknownOption,
		Name:      nameStr,
		Remaining: remaining,
		Err:       fmt.Errorf("unknown option: %s", nameStr),
	}
}

func flagAfterArgError(name any) error {
	nameStr := optionName(name)
	return &ParseError{
		Code: FlagUsedAfterArgs,
		Name: nameStr,
		Err:  fmt.Errorf("can't use %s after arguments", nameStr),
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

func listOfValues(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	case 2:
		return fmt.Sprintf("`%s' or `%s'", values[0], values[1])
	default:
		var b bytes.Buffer
		for i, v := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			if i == len(values)-1 {
				b.WriteString("or ")
			}
			b.WriteString("`")
			b.WriteString(v)
			b.WriteString("'")
		}
		return b.String()
	}
}

var _ error = (*InternalError)(nil)
var _ formattableError = errorTemplate{}
