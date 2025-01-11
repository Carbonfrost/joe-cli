package exec

import (
	"errors"
	"flag"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

// ArgList provides a value that describes a command and its arguments used as a flag, arg,
// or expression Value.  This uses the syntax
// that the find -exec expression uses, which is terminated by a semi-colon or plus sign.
// (e.g. -exec COMMAND ; or -exec COMMAND {} +).
type ArgList []string

type counter struct {
	seen bool
}

func (l *ArgList) Set(arg string) error {
	*l = append(*l, arg)
	return nil
}

func (l *ArgList) String() string {
	args, delim := l.split() // could have implied ;
	return strings.Join(append(args, delim), " ")
}

func (l *ArgList) NewCounter() cli.ArgCounter {
	return &counter{}
}

// Args gets the list of items added, including the command
// as Args()[0]
func (l *ArgList) Args() []string {
	args, _ := l.split()
	return args
}

// Command gets the command and its arguments
func (l *ArgList) Command() (string, []string) {
	args, _ := l.split()
	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

// split on the args and the delimiter (including
// if it is implied)
func (l *ArgList) split() ([]string, string) {
	res := *l
	if len(res) == 0 {
		return nil, ";"
	}
	c := res[len(res)-1]
	if c == ";" || c == "+" {
		return res[0 : len(res)-1], c
	}
	return res, ";"
}

// UsePlaceholder gets whether the arg list ended with +, which is used to
// indicate that the {} placeholder is to be expanded
func (l *ArgList) UsePlaceholder() bool {
	_, d := l.split()
	return d == "+"
}

func (ok *counter) Take(a string, possibleFlag bool) error {
	if ok.seen {
		return cli.EndOfArguments
	}

	ok.seen = (a == ";" || a == "+")
	return nil
}

func (ok *counter) Done() error {
	if ok.seen {
		return nil
	}
	return errors.New("must terminate expression with `;' or `+'")
}

var _ flag.Value = (*ArgList)(nil)
