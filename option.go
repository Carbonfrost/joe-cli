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

	maxOption

	None Option = 0
)

const (
	internalFlagHidden = internalFlags(1 << iota)
	internalFlagRequired
	internalFlagExits
)

var (
	optionMap = map[Option]ActionFunc{
		Hidden:   hiddenOption,
		Required: requiredOption,
		Exits:    wrapWithExit,
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
