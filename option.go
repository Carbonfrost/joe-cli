package cli

import (
	"fmt"
	"os"
)

type Option int
type internalFlags int

const (
	Hidden = Option(1 << iota)
	Required

	// Exits marks a flag that causes the app to exit.  This value is added to the options of flags
	// like --help and --version to indicate that they cause the app to exit when used.  It implicitly
	// causes the action of the flag to return SuccessStatus if they return no other error.  When this
	// option is set, it also causes the flag to be set apart visually on the help screen.
	Exits

	// MustExist indicates that when a File or Path value is used, it must represent a file
	// or path that exists.
	MustExist

	// SkipFlagParsing when applied to a Command causes everything submitted to the command
	// to be treated as if they are arguments.  Generally with this option, you must either define
	// an Arg that is a list that takes all the values or you must parse manually from the context
	// args
	SkipFlagParsing

	// DisallowFlagsAfterArgs option when used on a Command prevents flags from being specified after args.
	// To ease ergonomics, flags can be specified after arguments by default; however, to stop this, this
	// option can be set, which causes the first occurrence of an argument to indicate that the rest of the
	// command line consists only of arguments.  This option is implicitly set if any expressions or
	// sub-commands are defined.
	DisallowFlagsAfterArgs

	// WorkingDirectory labels a flag or argument as specifying the process working directory.
	// When used, the working directory will be changed to this path.
	WorkingDirectory

	// Optional makes the flag's value optional.  When this is set, the value of a flag is optional,
	// and the following values are set when the flag is present but its value is omitted:
	//
	//   integers (int, int32, int64, etc.)     1
	//   floats (float32, float64, etc.)        1.0
	//   time.Time                              time.Now()
	//   time.Duration                          1 second
	//   bool                                   true
	//   string                                 *
	//   []string                               *
	//   other Value                            *
	//
	//   * For string, []string, and any other Value implementation, using this option panics.
	//
	// This option is available to but not useful for bool because this is the default behavior
	// for bool.  If you need more customization, then you can use OptionalValue middleware.
	//
	Optional

	maxOption

	None Option = 0
)

const (
	internalFlagHidden = internalFlags(1 << iota)
	internalFlagRequired
	internalFlagExits
	internalFlagSkipFlagParsing
	internalFlagDisallowFlagsAfterArgs
)

var (
	optionMap = map[Option]ActionHandler{
		Hidden:                 ActionFunc(hiddenOption),
		Required:               ActionFunc(requiredOption),
		Exits:                  ActionFunc(wrapWithExit),
		MustExist:              Before(ActionFunc(mustExistOption)),
		SkipFlagParsing:        ActionFunc(skipFlagParsingOption),
		WorkingDirectory:       Before(ActionFunc(workingDirectoryOption)),
		Optional:               ActionFunc(optionalOption),
		DisallowFlagsAfterArgs: ActionFunc(disallowFlagsAfterArgsOption),
	}
)

func (o Option) wrap() *ActionPipeline {
	parts := splitOptions(int(o))
	if len(parts) == 0 {
		return nil
	}

	return &ActionPipeline{parts}
}

func (f internalFlags) hidden() bool {
	return f&internalFlagHidden == internalFlagHidden
}

func (f internalFlags) required() bool {
	return f&internalFlagRequired == internalFlagRequired
}

func (f internalFlags) exits() bool {
	return f&internalFlagExits == internalFlagExits
}

func (f internalFlags) skipFlagParsing() bool {
	return f&internalFlagSkipFlagParsing == internalFlagSkipFlagParsing
}

func (f internalFlags) disallowFlagsAfterArgs() bool {
	return f&internalFlagDisallowFlagsAfterArgs == internalFlagDisallowFlagsAfterArgs
}

func splitOptions(options int) []ActionHandler {
	var res []ActionHandler
	for current := 1; options != 0 && current < int(maxOption); current = current << 1 {
		if options&current == current {
			res = append(res, optionMap[Option(current)])
			options = options &^ current
		}
	}
	return res
}

func hiddenOption(c *Context) error {
	c.option().SetHidden()
	return nil
}

func requiredOption(c *Context) error {
	c.option().SetRequired()
	return nil
}

func wrapWithExit(c *Context) error {
	c.option().setInternalFlags(internalFlagExits)
	c.option().wrapAction(doThenExit)
	return nil
}

func mustExistOption(c *Context) error {
	v := c.Value("")
	if v == nil {
		return nil
	}
	f := v.(*File)
	if f.Exists() {
		return nil
	}
	_, err := f.Stat()
	return err
}

func skipFlagParsingOption(c *Context) error {
	c.target().setInternalFlags(internalFlagSkipFlagParsing)
	return nil
}

func workingDirectoryOption(c *Context) error {
	newDir := fmt.Sprint(c.Value(""))
	return os.Chdir(newDir)
}

func optionalOption(c *Context) error {
	c.Flag().setOptional()
	return nil
}

func disallowFlagsAfterArgsOption(c *Context) error {
	c.target().setInternalFlags(internalFlagDisallowFlagsAfterArgs)
	return nil
}
