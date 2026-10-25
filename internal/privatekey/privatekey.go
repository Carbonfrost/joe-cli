package privatekey

type T string

const (
	CommandNotFound   T = "__CommandNotFound"
	CompletionRequest T = "_CompletionRequest"
	DependsOn         T = "__DependsOn"
	OptionalAliases   T = "__OptionalAliases"
	OptionError       T = "_OptionError"
	PanicData         T = "__PanicData"
	PseudoCommand     T = "__PseudoCommand"
	PseudoFlag        T = "__PseudoFlag"
	ShellCompletes    T = "__ShellCompletes"
	Synopsis          T = "_Synopsis"
	Validator         T = "__Validator"
	ValueHelpText     T = "_ValueHelpText"
)
