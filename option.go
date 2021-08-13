package cli

type Option int
type internalFlags int

const (
	Hidden = Option(1 << iota)
	Required
	maxOption

	None Option = 0
)

const (
	internalFlagHidden = internalFlags(1 << iota)
	internalFlagRequired
)

var (
	optionMap = map[Option]ActionFunc{
		Hidden:   hiddenOption,
		Required: requiredOption,
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
