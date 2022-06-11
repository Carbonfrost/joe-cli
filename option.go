package cli

import (
	"encoding"
	"fmt"
	"io"
	"math/bits"
	"os"
	"sort"
	"strings"
)

// Option provides a built-in convenience configuration for flags, args, and commands.
type Option int

type Feature interface {
	~int
}

// FeatureMap provides a map from a feature identifier to an action.  A common idiom within
// joe-cli is to define a bitmask representing
type FeatureMap[T Feature] map[T]Action

type internalFlags uint32

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
	//   time.Duration                          1 second
	//   bool                                   true
	//   string                                 *
	//   []string                               *
	//   []byte                                 *
	//   other Value                            *
	//
	//   * For string, []string, []byte, and any other Value implementation, using this option panics.
	//
	// For short options, no space can be between the flag and value (e.g. you need -sString to
	// specify a String to the -s option).
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
	// be strategic merging.  This pertains to strings, list, fileset, and map flags.  By default,
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

	// PreventSetup is used to opt-out of any setup actions.  Typically,
	// reusable actions provide implicit setup such as choosing a good default
	// name or environment variables.  To stop this, add PreventSetup to
	// the pipeline which will taint the implicit setup actions causing them
	// to be skipped.  However, note that preventing setup could cause the
	// flag, arg, or command not to work properly if any required
	// configurations go missing. PreventSetup only applies in the Uses
	// pipeline and to actions in the Uses pipeline.  The action is recursive to
	// the scope of the app, flag, arg, or command.
	PreventSetup

	// EachOccurrence causes the Action for a flag or arg to be called once for each occurrence
	// rather than just once for the winning value.  This makes sense for flags or args
	// where each occurrence is treated as distinct whereas by default, clients are usually
	// only concerned with the last occurrence or the aggregated value.  For example, for the
	// command "app -a first -a last", the value for -a is "last" (or "first last" if -a is set to Merge);
	// however, if EachOccurrence were specified, then the
	// Action associated with -a will be called twice with the context value set equal to the corresponding
	// occurrence.  Note that only the default lookup will contain the occurrence; other lookups will
	// use the winning value.  (i.e. for the example, String("") will vary as "first" and "last" in the
	// two corresponding Action calls; however String("a") will always be "last").  This also applies to
	// Raw("").
	EachOccurrence

	// FileReference indicates that the flag or argument is a reference to a file which is loaded
	// and whose contents provide the actual value of the flag.
	FileReference

	// AllowFileReference allows a flag or argument to use the special syntax @file to mean that
	// the value is obtained by loading the contents of a file rather than directly.  When the
	// plain syntax without @ is used, the value is taken as the literal contents of an unnamed
	// file.
	AllowFileReference

	// SortedFlags causes flags to be sorted on the help screen generated for the command or app.
	SortedFlags

	// SortedFlags causes sub-commands to be sorted on the help screen generated for the command or app.
	SortedCommands

	maxOption

	// None represents no options
	None Option = 0

	// Sorted causes flags and sub-commands to be sorted on the help screen generated for the command or app.
	Sorted = SortedFlags | SortedCommands
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
	internalFlagFlagOnly   // true for Flag without an argument
	internalFlagOptional   // true if value is optional
	internalFlagPersistent // true when the option is a clone of a peristent parent flag
	internalFlagDestinationImplicitlyCreated
)

var (
	builtinOptions = FeatureMap[Option]{
		Hidden:                 ActionFunc(hiddenOption),
		Required:               ActionFunc(requiredOption),
		Exits:                  ActionFunc(wrapWithExit),
		MustExist:              Before(ActionFunc(mustExistOption)),
		SkipFlagParsing:        setInternalFlag(internalFlagSkipFlagParsing),
		WorkingDirectory:       Before(ActionFunc(workingDirectoryOption)),
		Optional:               ActionFunc(optionalOption),
		DisallowFlagsAfterArgs: setInternalFlag(internalFlagDisallowFlagsAfterArgs),
		No:                     ActionFunc(noOption),
		NonPersistent:          ActionFunc(nonPersistentOption),
		DisableSplitting:       ActionFunc(disableSplittingOption),
		Merge:                  setInternalFlag(internalFlagMerge),
		RightToLeft:            setInternalFlag(internalFlagRightToLeft),
		PreventSetup:           ActionOf((*Context).PreventSetup),
		EachOccurrence:         ActionFunc(eachOccurrenceOpt),
		AllowFileReference:     ActionFunc(allowFileReferenceOpt),
		FileReference:          ActionFunc(fileReferenceOpt),
		SortedFlags:            Before(ActionFunc(sortedFlagsOpt)),
		SortedCommands:         Before(ActionFunc(sortedCommandsOpt)),
	}

	builtinOptionLabels = map[Option]string{
		Hidden:                 "HIDDEN",
		Required:               "REQUIRED",
		Exits:                  "EXITS",
		MustExist:              "MUST_EXIST",
		SkipFlagParsing:        "SKIP_FLAG_PARSING",
		WorkingDirectory:       "WORKING_DIRECTORY",
		Optional:               "OPTIONAL",
		DisallowFlagsAfterArgs: "DISALLOW_FLAGS_AFTER_ARGS",
		No:                     "NO",
		NonPersistent:          "NON_PERSISTENT",
		DisableSplitting:       "DISABLE_SPLITTING",
		Merge:                  "MERGE",
		RightToLeft:            "RIGHT_TO_LEFT",
		PreventSetup:           "PREVENT_SETUP",
		EachOccurrence:         "EACH_OCCURRENCE",
		AllowFileReference:     "ALLOW_FILE_REFERENCE",
		FileReference:          "FILE_REFERENCE",
		SortedFlags:            "SORTED_FLAGS",
		SortedCommands:         "SORTED_COMMANDS",
	}
)

func (o Option) String() string {
	var res []string
	splitOptionsHO(o, func(current Option) {
		name := builtinOptionLabels[current]
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
	for k, v := range builtinOptionLabels {
		if token == v {
			return k
		}
	}
	return None
}

// Execute treats the options as if an action
func (o Option) Execute(c *Context) (err error) {
	return builtinOptions.Pipeline(o).Execute(c)
}

func (m FeatureMap[T]) Pipeline(values T) Action {
	var (
		i     int
		parts []Action
	)

	// Sort options in order of hamming weight so that any composite flag
	// is invoked before single flags.
	keys := make([]uint, len(m))
	for k := range m {
		keys[i] = uint(k)
		i++
	}
	sort.Slice(keys, func(i, j int) bool {
		return bits.OnesCount(keys[i]) > bits.OnesCount(keys[j])
	})
	options := uint(values)

	for _, current := range keys {
		if options&current == current {
			action := m[T(current)]
			parts = append(parts, action)
			options = options &^ current
		}
	}

	if len(parts) == 0 {
		return ActionOf(nil)
	}

	return ActionPipeline(parts)
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

func (f internalFlags) flagOnly() bool {
	return f&internalFlagFlagOnly == internalFlagFlagOnly
}

func (f internalFlags) optional() bool {
	return f&internalFlagOptional == internalFlagOptional
}

func (f internalFlags) persistent() bool {
	return f&internalFlagPersistent == internalFlagPersistent
}

func (f internalFlags) destinationImplicitlyCreated() bool {
	return f&internalFlagDestinationImplicitlyCreated == internalFlagDestinationImplicitlyCreated
}

func (f internalFlags) toRaw() RawParseFlag {
	var flags RawParseFlag
	if f.disallowFlagsAfterArgs() {
		flags |= RawDisallowFlagsAfterArgs
	}
	if f.rightToLeft() {
		flags |= RawRTL
	}
	return flags
}

func splitOptionsHO(opts Option, fn func(Option)) {
	options := int(opts)
	for current := 1; options != 0 && current < int(maxOption); current = current << 1 {
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
	return c.Do(AtTiming(ActionOf(doThenExit), ActionTiming))
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

		// If the file name was blank, we ignore this error because
		// it implies a blank value was set and the user must handle
		return ignoreBlankPathError(err)
	}
	return fmt.Errorf("file not found: %v", v)
}

func workingDirectoryOption(c *Context) error {
	if c.Seen("") {
		newDir := fmt.Sprint(c.Value(""))
		if newDir == "" {
			return nil
		}
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
	_ = syn.withLongAndShort(
		[]string{"[no-]" + syn.Long},
		syn.Shorts,
	)
	wrapAction := func(v Action) ActionFunc {
		return func(c *Context) error {
			return execute(c.copy(
				&wrapLookupContext{
					optionContext: c.internal.(*optionContext),
					actual:        f,
				},
				false,
			), v)
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
			return execute(c, wrapAction(ActionOf(f.Action)))
		},
	})
	return nil
}

func eachOccurrenceOpt(c1 *Context) error {
	return c1.Do(AtTiming(middlewareFunc(func(c *Context, next Action) error {
		opt := func() *internalOption {
			switch o := c.option().(type) {
			case *Flag:
				return &o.option
			case *Arg:
				return &o.option
			}
			return nil
		}()
		start := opt.value
		mini := &wrapOccurrenceContext{
			optionContext: c.internal.(*optionContext),
		}

		scope := c.copy(mini, true)

		// Obtain either the zero value or Reset() the value
		start = start.cloneZero()
		for i := 0; i < mini.numOccurs(); i++ {
			mini.index = i

			// Pretend this is the first occurrence
			start.applyValueConventions(opt.flags, 1)
			for _, s := range mini.current() {
				start.Set(s, opt)
			}
			mini.val = start.p

			if err := next.Execute(scope); err != nil {
				return err
			}
		}
		return nil
	}), ActionTiming))
}

func allowFileReferenceOpt(c *Context) error {
	f := c.FS
	return c.Do(Transform(func(raw []string) (interface{}, error) {
		readers := make([]io.Reader, len(raw)-1)
		for i, s := range raw[1:] {
			if strings.HasPrefix(s, "@") {
				f, err := f.Open(s[1:])
				if err != nil {
					return nil, err
				}
				readers[i] = f
			} else {
				readers[i] = strings.NewReader(s)
			}
		}
		return io.MultiReader(readers...), nil
	}))
}

func fileReferenceOpt(c *Context) error {
	f := c.FS
	return c.Do(Transform(func(raw []string) (interface{}, error) {
		readers := make([]io.Reader, len(raw)-1)
		for i, s := range raw[1:] {
			f, err := f.Open(s)
			if err != nil {
				return nil, err
			}
			readers[i] = f
		}
		return io.MultiReader(readers...), nil
	}))
}

func sortedFlagsOpt(c *Context) error {
	cmd := c.Command()
	sort.Sort(FlagsByName(cmd.Flags))
	return nil
}

func sortedCommandsOpt(c *Context) error {
	cmd := c.Command()
	sort.Sort(CommandsByName(cmd.Subcommands))
	return nil
}

var (
	_ encoding.TextMarshaler   = (Option)(0)
	_ encoding.TextUnmarshaler = (*Option)(nil)
	_ Action                   = (*Option)(nil)
)
