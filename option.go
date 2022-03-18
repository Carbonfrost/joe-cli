package cli

import (
	"encoding"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// Option provides a built-in convenience configuration for flags, args, and commands.
type Option int
type internalFlags int
type userOption uint32
type optionDef struct {
	Action Action
	Name   string
}

const (
	// Hidden causes the option to be Hidden
	Hidden = Option(1 << iota)

	// Required marks a flag as required.  When set, an error is generated if the flag is  not
	// specified
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

	// No introduces a mirror flag to a Boolean flag that provides the false value.
	// When set as the option on a flag, this causes a mirror flag to be created with the
	// prefix no-.  For example, for the flag --color, --no-color gets generated.  In addition,
	// the mirror flag is hidden and the help screen provides a concise summary.  The mirror
	// flag inherits the Before, After, and Action middleware pipelines; however, the Value
	// for the flag in the context always matches the original flag.
	No

	// NonPersistent marks a flag as being non-persistent.  By default, any flag defined by an ancestor
	// command can be set by a sub-command.  However, when present, the NonPersistent option causes this to
	// be treated as a usage error.
	NonPersistent

	// DisableSplitting disable splitting on commas.  This affects List arguments and flags.
	// By default, commas are used as delimiters to split list values.
	// For example --planet Mars,Jupiter is equivalent to --planet Mars --planet Jupiter by default;
	// however, by enabling this option, such splitting is disabled.  For custom implementations of
	// flag Value, you can implement this method DisableSplitting() to hook into this option.
	DisableSplitting

	// Merge when set, indicates that rather than overwrite values set in list, there should
	// be strategic merging.  This pertains to list, fileset, and map flags.  By default,
	// the value that is used to initialize any of these flags is treated as a default which
	// is overwritten if the user specifies an environment variable or any value.  To stop this,
	// the Merge option can be used
	Merge

	// RightToLeft causes arguments to bind right-to-left instead of left-to-right.
	// This option applies to commands and expressions, and it affects how optional args are
	// interpreted when there are fewer values than are optional.  By default, with left-to-right binding,
	// in this situation, the leftmost optional args are filled first, leaving any subsequent args empty.
	// However, with right-to-left binding, we fill as many rightmost args as there are available
	// values.  For example, if arg a and b are optional (as in, Arg.NArg set to 0) and a subsequent
	// arg r exists, then if there is one value passed to the command, only r will be set.  If there
	// are two values, both b and r will be set.
	// Note that despite its name, args still bind in order.
	//
	RightToLeft

	maxOption

	// None represents no options
	None Option = 0
)

const (
	internalFlagHidden = internalFlags(1 << iota)
	internalFlagRequired
	internalFlagExits
	internalFlagSkipFlagParsing
	internalFlagDisallowFlagsAfterArgs
	internalFlagNonPersistent
	internalFlagDisableSplitting
	internalFlagMerge
	internalFlagRightToLeft
)

var (
	builtinOptions = map[Option]optionDef{
		Hidden: {
			Action: ActionFunc(hiddenOption),
			Name:   "HIDDEN",
		},
		Required: {
			Action: ActionFunc(requiredOption),
			Name:   "REQUIRED",
		},
		Exits: {
			Action: ActionFunc(wrapWithExit),
			Name:   "EXITS",
		},
		MustExist: {
			Action: Before(ActionFunc(mustExistOption)),
			Name:   "MUST_EXIST",
		},
		SkipFlagParsing: {
			Action: setInternalFlag(internalFlagSkipFlagParsing),
			Name:   "SKIP_FLAG_PARSING",
		},
		WorkingDirectory: {
			Action: Before(ActionFunc(workingDirectoryOption)),
			Name:   "WORKING_DIRECTORY",
		},
		Optional: {
			Action: ActionFunc(optionalOption),
			Name:   "OPTIONAL",
		},
		DisallowFlagsAfterArgs: {
			Action: setInternalFlag(internalFlagDisallowFlagsAfterArgs),
			Name:   "DISALLOW_FLAGS_AFTER_ARGS",
		},
		No: {
			Action: ActionFunc(noOption),
			Name:   "NO",
		},
		NonPersistent: {
			Action: ActionFunc(nonPersistentOption),
			Name:   "NON_PERSISTENT",
		},
		DisableSplitting: {
			Action: ActionFunc(disableSplittingOption),
			Name:   "DISABLE_SPLITTING",
		},
		Merge: {
			Action: setInternalFlag(internalFlagMerge),
			Name:   "MERGE",
		},
		RightToLeft: {
			Action: setInternalFlag(internalFlagRightToLeft),
			Name:   "RIGHT_TO_LEFT",
		},
	}

	userOptions = map[Option]optionDef{}

	// Custom options start at 1 << 24, exclusive, which must be higher than built-in
	startUserOption            = 24
	globalOption    userOption = userOption(startUserOption)
)

// NewOption allocates a custom user option.  This is typically used by add-ons.
func NewOption(name string, action Action) Option {
	for k, v := range userOptions {
		if name == v.Name {
			return k
		}
	}
	val := Option(1 << globalOption.inc())
	userOptions[val] = optionDef{
		Action: action,
		Name:   name,
	}

	return val
}

func getOptionDef(o Option) optionDef {
	if res, ok := builtinOptions[o]; ok {
		return res
	}
	return userOptions[o]
}

func (o Option) String() string {
	var res []string
	splitOptionsHO(o, func(current Option) {
		name := getOptionDef(current).Name
		if name != "" {
			res = append(res, name)
		}
	})
	return strings.Join(res, ", ")
}

// MarshalText provides the textual representation
func (o Option) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

// UnmarshalText converts the textual representation
func (o *Option) UnmarshalText(b []byte) error {
	res := *o
	for _, s := range strings.Split(string(b), ",") {
		token := strings.TrimSpace(s)
		res |= unmarshalText(token)
	}
	*o = res
	return nil
}

func unmarshalText(token string) Option {
	for k, v := range builtinOptions {
		if token == v.Name {
			return k
		}
	}
	for k, v := range userOptions {
		if token == v.Name {
			return k
		}
	}
	return None
}

func (o Option) wrap() *ActionPipeline {
	var parts []Action
	splitOptionsHO(o, func(current Option) {
		action := getOptionDef(current).Action
		parts = append(parts, action)
	})
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

func (f internalFlags) nonPersistent() bool {
	return f&internalFlagNonPersistent == internalFlagNonPersistent
}

func (f internalFlags) disableSplitting() bool {
	return f&internalFlagDisableSplitting == internalFlagDisableSplitting
}

func (f internalFlags) merge() bool {
	return f&internalFlagMerge == internalFlagMerge
}

func (f internalFlags) rightToLeft() bool {
	return f&internalFlagRightToLeft == internalFlagRightToLeft
}

func (u *userOption) inc() uint32 {
	return atomic.AddUint32((*uint32)(u), 1)
}

func splitOptionsHO(opts Option, fn func(Option)) {
	options := int(opts)
	for current := 1; options != 0 && current < int(maxOption); current = current << 1 {
		if options&current == current {
			fn(Option(current))
			options = options &^ current
		}
	}

	for index := startUserOption; options != 0 && index <= int(globalOption); index++ {
		current := 1 << index
		if options&current == current {
			fn(Option(current))
			options = options &^ current
		}
	}
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

func nonPersistentOption(c *Context) error {
	c.option().setInternalFlags(internalFlagNonPersistent)
	return nil
}

func disableSplittingOption(c *Context) error {
	c.option().setInternalFlags(internalFlagDisableSplitting)
	return nil
}

func mustExistOption(c *Context) error {
	v := c.Value("")
	if v == nil {
		return nil
	}
	f := v.(fileExists)
	if f.Exists() {
		return nil
	}
	if s, ok := v.(fileStat); ok {
		_, err := s.Stat()
		return err
	}
	return fmt.Errorf("file not found: %v", v)
}

func workingDirectoryOption(c *Context) error {
	if c.Seen("") {
		newDir := fmt.Sprint(c.Value(""))
		return os.Chdir(newDir)
	}
	return nil
}

func optionalOption(c *Context) error {
	c.Flag().setOptional()
	return nil
}

func setInternalFlag(f internalFlags) ActionFunc {
	return func(c *Context) error {
		c.target().setInternalFlags(f)
		return nil
	}
}

func noOption(c *Context) error {
	f := c.Flag()

	syn := f.synopsis()
	syn.long = "[no-]" + syn.long
	wrapAction := func(v Action) ActionFunc {
		return func(c *Context) error {
			return execute(v, c.copy(
				&wrapLookupContext{
					flagContext: c.internal.(*flagContext),
					actual:      f,
				},
				false,
			))
		}
	}

	cmd := c.Command()
	cmd.Flags = append(cmd.Flags, &Flag{
		HelpText:  f.HelpText,
		UsageText: f.UsageText,
		Name:      "no-" + f.Name,
		Category:  f.Category,
		Value:     Bool(),
		Options:   Hidden,
		Before:    wrapAction(ActionOf(f.Before)),
		After:     wrapAction(ActionOf(f.After)),
		Action: func(c *Context) error {
			f.Set("false")
			return execute(wrapAction(ActionOf(f.Action)), c)
		},
	})
	return nil
}

var _ encoding.TextMarshaler = (Option)(0)
var _ encoding.TextUnmarshaler = (*Option)(nil)
