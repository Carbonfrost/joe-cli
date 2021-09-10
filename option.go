package cli

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

	maxOption

	None Option = 0
)

const (
	internalFlagHidden = internalFlags(1 << iota)
	internalFlagRequired
	internalFlagExits
	internalFlagSkipFlagParsing
)

var (
	optionMap = map[Option]ActionFunc{
		Hidden:          hiddenOption,
		Required:        requiredOption,
		Exits:           wrapWithExit,
		MustExist:       mustExistOption,
		SkipFlagParsing: skipFlagParsingOption,
	}
)

func (o Option) Execute(c *Context) error {
	parts := splitOptions(int(o))
	pipe := &ActionPipeline{parts}
	return pipe.Execute(c)
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
