package cli

import (
	"fmt"
)

type ErrorCode int

type Error struct {
	Code  ErrorCode
	Err   error
	Name  string
	Value interface{}
}

type ExitCoder interface {
	error
	ExitCode() int
}

type exitError struct {
	message  interface{}
	exitCode int
}

const (
	UnexpectedArgument = ErrorCode(iota)
	CommandNotFound
	UnknownOption
	MissingArgument
	InvalidArgument
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
			return &Error{
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

func (e *Error) ExitCode() int {
	return 2
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Code.String()
	}
	return e.Err.Error()
}

func (e ErrorCode) String() string {
	switch e {
	case UnexpectedArgument:
		return "unexpected argument"
	case CommandNotFound:
		return "is not a command"
	case UnknownOption:
		return "unknown option"
	case MissingArgument:
		return "missing parameter"
	case InvalidArgument:
		return "parameter not valid "
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
	return &Error{
		Code: CommandNotFound,
		Err:  fmt.Errorf("%q is not a command", name),
		Name: name,
	}
}

func unexpectedArgument(value string) *Error {
	return &Error{
		Code: UnexpectedArgument,
		Err:  fmt.Errorf("too many arguments: %q", value),
	}
}

func unknownOption(name interface{}) error {
	nameStr := func() string {
		switch n := name.(type) {
		case rune:
			if n == '-' {
				return "-"
			} else {
				return "-" + string(n)
			}
		case string:
			return "--" + n
		}
		panic("unreachable!")
	}()

	return &Error{
		Code: UnknownOption,
		Name: nameStr,
		Err:  fmt.Errorf("unknown option: %s", nameStr),
	}
}

func missingArgument(o *internalOption) error {
	return &Error{
		Code: MissingArgument,
		Name: o.Name(),
		Err:  fmt.Errorf("missing parameter for %s", o.Name()),
	}
}

func setFlagError(o *internalOption, value string, err error) error {
	return &Error{
		Code:  InvalidArgument,
		Name:  o.Name(),
		Value: value,
		Err:   err,
	}
}
