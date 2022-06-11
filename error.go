package cli

import (
	"errors"
	"fmt"
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
	// caused the rorr
	Name string

	// Value is the value that caused the error
	Value string

	// Remaining contains arguments which could not be parsed
	Remaining []string
}

// ExitCoder is an error that knows how to convert to its exit code
type ExitCoder interface {
	error

	// ExitCode obtains the exit code for the error
	ExitCode() int
}

type exitError struct {
	message  interface{}
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
)

// Exit formats an error message using the default formats for each of the arguments,
// except the last one, which is interpreted as the desired exit code.  The function
// provides similar semantics to fmt.Sprint in that all values are converted to text
// and joined together.  Spaces are added between operands when neither is a string.
// If the last argument is an integer, it is interpreted as the exit code that will
// be generated when the program exits.  If no integer is present, the value 1 is used.
func Exit(message ...interface{}) ExitCoder {
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
	}
	return "unknown error"
}

func (e *exitError) Error() string {
	return fmt.Sprintf("%v", e.message)
}

func (e *exitError) ExitCode() int {
	return e.exitCode
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
	msg := "expected argument"
	if count > 1 {
		msg = fmt.Sprintf("expected %d arguments", count)
	}
	return &ParseError{
		Code: ExpectedArgument,
		Err:  errors.New(msg),
	}
}

func unknownOption(name interface{}, remaining []string) error {
	nameStr := optionName(name)
	return &ParseError{
		Code:      UnknownOption,
		Name:      nameStr,
		Remaining: remaining,
		Err:       fmt.Errorf("unknown option: %s", nameStr),
	}
}

func flagAfterArgError(name interface{}) error {
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

func unknownExpr(name string) error {
	return &ParseError{
		Code: UnknownExpr,
		Name: name,
		Err:  fmt.Errorf("unknown expression: %s", name),
	}
}

func argsMustPrecedeExprs(arg string) error {
	return &ParseError{
		Code:  ArgsMustPrecedeExprs,
		Value: arg,
		Err:   fmt.Errorf("arguments must precede expressions: %q", arg),
	}
}

func optionName(name interface{}) string {
	switch n := name.(type) {
	case rune:
		if n == '-' {
			return "-"
		}
		return "-" + string(n)
	case string:
		return "--" + n
	}
	panic("unreachable!")
}
