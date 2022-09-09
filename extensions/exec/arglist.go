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
	return strings.Join([]string(*l), " ")
}

func (l *ArgList) NewCounter() cli.ArgCounter {
	return &counter{}
}

// Args gets the list of items added.
func (l *ArgList) Args() []string {
	res := *l
	return res[0 : len(res)-1]
}

// UsePlaceholder gets whether the arg list ended with +, which is used to
// indicate that the {} placeholder is to be expanded
func (l *ArgList) UsePlaceholder() bool {
	res := *l
	return res[len(res)-1] == "+"
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
