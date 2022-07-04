package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"strings"
	"time"
)

// ActionFunc provides the basic function for an Action
type ActionFunc func(*Context) error

//counterfeiter:generate . Action

// Action represents the building block of the various actions
// to perform when an app, command, flag, or argument is being evaluated.
type Action interface {

	// Execute will execute the action.  If the action returns an error, this
	// may cause subsequent actions in the pipeline not to be run and cause
	// the app to exit with an error exit status.
	Execute(*Context) error
}

//counterfeiter:generate . Middleware

// Middleware provides an action which controls how and whether the next
// action in the pipeline is executed
type Middleware interface {
	Action

	// ExecuteWithNext will execute the action and invoke Execute on the next
	// action
	ExecuteWithNext(*Context, Action) error
}

// ActionPipeline represents an action composed of several steps.  To create
// this value, use the Pipeline function
type ActionPipeline []Action

// Setup provides simple initialization, typically used in Uses pipeline.  The Setup action
// will add the specified actions to the Before, main, and After action and run
// the Uses action immediately.
type Setup struct {
	Uses   interface{}
	Before interface{}
	Action interface{}
	After  interface{}

	// Optional causes setup to ignore timing errors.  By default, Setup depends upon the
	// timing, which means that an error will occur if Setup is used within a timing context
	// that is later than the corresponding pipelines.  For example, if you have Setup.Uses set,
	// this implies that the Setup can only be itself added to a Uses pipeline; if you add it
	// to a Before, Action, or After pipeline, an error will occur because it is too late to process
	// the pipeline set in Setup.Uses.  Setting Optional to true will prevent this error.
	//
	// The common use case for this is to allow defining new actions by simply returning a Setup, and letting
	// the user of the new action decide which parts of Setup get used by allowing them to specify the
	// setup in the pipeline of their choice.  In the example, if the user
	// assigned the action in the Action pipeline, this would imply that they don't care about the
	// initialization behavior the action provides.  If the initialization is genuinely optional
	// (and not a usage error of the new action), it is appropriate to set Optional to true.
	Optional bool
}

// Prototype implements an action which sets up a flag or arg.  The
// prototype copies its values to the corresponding flag or arg if they have not
// already been set.  Some values are merged rather than overwritten:
// Data, Options, EnvVars, and Aliases.
// If setup has been prevented with the PreventSetup action,
// the protoype will do nothing.  The main use of prototype is in extensions to provide
// reasonable defaults
type Prototype struct {
	Aliases     []string
	Category    string
	Data        map[string]interface{}
	DefaultText string
	Description string
	EnvVars     []string
	FilePath    string
	HelpText    string
	ManualText  string
	Name        string
	Options     Option
	UsageText   string
	Value       interface{}
	Setup       Setup
	Completion  Completion
}

// ValidatorFunc defines an Action that applies a validation rule to
// the explicit raw occurrence values for a flag or argument.
type ValidatorFunc func(s []string) error

type hookable interface {
	hookAfter(pattern string, handler Action) error
	hookBefore(pattern string, handler Action) error
}

type customizable interface {
	customize(pattern string, handler Action)
	customizations() []*hook
}

type target interface {
	appendAction(Timing, Action)
	setDescription(string)
	setHelpText(string)
	setManualText(string)
	setCategory(name string)
	SetData(name string, v interface{})
	LookupData(name string) (interface{}, bool)
	setInternalFlags(internalFlags)
	internalFlags() internalFlags
	completion() Completion
}

type hooksSupport struct {
	before []*hook
	after  []*hook
}

type customizableSupport struct {
	items []*hook
}

type pipelinesSupport struct {
	p *actionPipelines
}

type actionPipelines struct {
	Initializers Action // Must be strictly initializers (no automatic regrouping)
	Before       beforePipeline
	Action       Action
	After        Action
}

type withTimingWrapper struct {
	Action
	t Timing
}

type cons struct {
	action Action
	next   *cons
}

type middlewareFunc func(*Context, Action) error

// beforePipeline has sub-timings, defined in actualBeforeIndex map
type beforePipeline [4]ActionPipeline

type panicData struct {
	recovered string
	stack     string
}

// Timing enumerates the timing of an action
type Timing int

const (
	// InitialTiming which occurs during the Uses pipeline
	InitialTiming Timing = iota
	// BeforeTiming which occurs before the command executes
	BeforeTiming
	// ActionTiming which occurs for the primary action
	ActionTiming
	// AfterTiming which occurs after the command executes
	AfterTiming

	syntheticTiming

	// ValidatorTiming represents timing that happens when values are being validated
	// for an arg or flag.  This timing can be set with the AtTiming function which affects
	// the sort order of actions so that validation occurs before all other actions in Before
	// pipeline.  When the action runs, the actual timing will be BeforeTiming.
	ValidatorTiming

	// ImplicitValueTiming represents timing that happens when an implied value is being
	// computed for an arg or flag.  This timing can be set with the AtTiming function
	// which affects the sort order of actions so that implied value timing occurs just before
	// the action.  When the action runs, the actual timing will be BeforeTiming.
	ImplicitValueTiming

	// justBeforeTiming is internally used for actions that must happen just before the
	// Action timing
	justBeforeTiming
)

const (
	implicitTimingEnabledKey = "__ImplicitTimingEnabled"
	panicDataKey             = "__PanicData"
)

var (
	emptyAction Action = ActionFunc(emptyActionImpl)
	valueType          = reflect.TypeOf((*Value)(nil)).Elem()

	actualBeforeIndex = map[Timing]int{
		ValidatorTiming:     0,
		BeforeTiming:        1,
		ImplicitValueTiming: 2,
		justBeforeTiming:    3,
	}

	defaultCommand = actionPipelines{
		Initializers: Pipeline(
			rootCommandInitializers(
				Pipeline(
					ActionFunc(setupDefaultIO),
					ActionFunc(setupDefaultData),
					ActionFunc(setupDefaultTemplateFuncs),
					ActionFunc(setupDefaultTemplates),
					ActionFunc(optionalCommand("help", defaultHelpCommand)),
					ActionFunc(optionalFlag("help", defaultHelpFlag)),
					ActionFunc(optionalCommand("version", defaultVersionCommand)),
					ActionFunc(optionalFlag("version", defaultVersionFlag)),
					ActionFunc(setupCompletion),
				),
			),
			ActionFunc(ensureSubcommands),
			ActionFunc(initializeFlagsArgs),
			ActionFunc(initializeSubcommands),
			ActionFunc(handleCustomizations),
		),
		Action: Pipeline(
			ActionFunc(triggerRobustParsingAndCompletion),
		),
		Before: beforePipeline{
			nil,
			Pipeline(
				ActionFunc(triggerBeforeFlags),
				ActionFunc(triggerBeforeArgs),
			),
			nil,
		},
		After: Pipeline(
			ActionFunc(triggerAfterArgs),
			ActionFunc(triggerAfterFlags),
			ActionFunc(failWithContextError),
		),
	}

	defaultOption = actionPipelines{
		Initializers: Pipeline(
			ActionFunc(setupValueInitializer),
			ActionFunc(setupOptionFromEnv),
			ActionFunc(fixupOptionInternals),
			ActionFunc(handleCustomizations),
		),
	}

	cantHookError = errors.New("hooks are not supported in this context")

	defaultHelpCommand = &Command{
		Name:     "help",
		HelpText: "Display help for a command",
		Args: []*Arg{
			{
				Name:  "command",
				Value: List(),
				NArg:  -1,
			},
		},
		Action: displayHelp,
	}

	defaultHelpFlag = &Flag{
		Name:     "help",
		HelpText: "Display this help screen then exit",
		Value:    Bool(),
		Options:  Exits,
		Action:   displayHelp,
	}

	defaultVersionFlag = &Flag{
		Name:     "version",
		HelpText: "Print the build version then exit",
		Value:    Bool(),
		Options:  Exits,
		Action:   PrintVersion(),
	}

	defaultVersionCommand = &Command{
		Name:     "version",
		HelpText: "Print the build version then exit",
		Action:   PrintVersion(),
	}
)

// Execute executes the Setup, which assignes the various parts to their
// pipelines
func (s Setup) Execute(c *Context) error {
	if s.Uses != nil {
		if err := c.act(s.Uses, InitialTiming, s.Optional); err != nil {
			return err
		}
	}
	if s.Before != nil {
		if err := c.Before(s.Before); err != nil {
			return err
		}
	}
	if err := c.Action(s.Action); err != nil {
		return err
	}
	if err := c.After(s.After); err != nil {
		return err
	}
	return nil
}

// Pipeline combines various actions into a single action.  Compared to using
// ActionPipeline directly, the actions are flattened if any nested pipelines are
// present.
func Pipeline(actions ...interface{}) ActionPipeline {
	myActions := make([]Action, 0, len(actions))
	for _, a := range actions {
		if pipe, ok := a.(ActionPipeline); ok {
			myActions = append(myActions, pipe.flatten()...)
			continue
		}
		myActions = append(myActions, ActionOf(a))
	}

	return myActions
}

// SuppressError wraps an action to ignore its error.
func SuppressError(a Action) Action {
	return ActionFunc(func(c *Context) error {
		a.Execute(c)
		return nil
	})
}

// Recover wraps an action to recover from a panic
func Recover(a Action) Action {
	return ActionFunc(func(c *Context) error {
		defer func() {
			if rvr := recover(); rvr != nil {
				c.SetData(panicDataKey, &panicData{
					recovered: fmt.Sprint(rvr),
					stack:     formatStack(),
				})
			}
		}()
		return a.Execute(c)
	})
}

func formatStack() string {
	return string(debug.Stack())
}

func failWithContextError(c *Context) error {
	if r, ok := c.LookupData(panicDataKey); ok {
		rvr := r.(*panicData)
		fmt.Fprintf(c.Stderr, rvr.stack)
		return fmt.Errorf(rvr.recovered)
	}
	return nil
}

// Before revises the timing of the action so that it runs in the Before pipeline.
// This function is used to wrap actions in the initialization pipeline that will be
// deferred until later.
func Before(a Action) Action {
	return AtTiming(a, BeforeTiming)
}

// After revises the timing of the action so that it runs in the After pipeline.
// This function is used to wrap actions in the initialization pipeline that will be
// deferred until later.
func After(a Action) Action {
	return AtTiming(a, AfterTiming)
}

// Initializer marks an action handler as being for the initialization phase.  When such a handler
// is added to the Uses pipeline, it will automatically be associated correctly with the initialization
// of the value.  Otherwise, this handler is not special
func Initializer(a Action) Action {
	return AtTiming(a, InitialTiming)
}

// AtTiming wraps an action and causes it to execute at the given timing.
func AtTiming(a Action, t Timing) Action {
	return withTiming(a, t)
}

func withTiming(a Action, t Timing) Action {
	return withTimingWrapper{a, t}
}

// ActionOf converts a value to an Action.  Any of the following types can be converted:
//
//   * func(*Context) error  (same signature as Action.Execute)
//   * func(*Context)
//   * func(context.Context) error
//   * func(context.Context)
//   * func() error
//   * func()
//   * Action
//
// As a special case, these signatures are allowed in order to provide middleware:
//
//    * func(Action)Action
//    * func(*Context, Action) error
//
// Remember that the next action can be nil, and indeed the implementation of
// Execute (for implementing plain Action) the approach is to delegate to this function
// using a nil next action.
//
// Any other type causes a panic.
func ActionOf(item interface{}) Action {
	switch a := item.(type) {
	case nil:
		return emptyAction
	case func(*Context) error:
		return ActionFunc(a)
	case Action:
		return a
	case func(*Context):
		return ActionFunc(func(c *Context) error {
			a(c)
			return nil
		})
	case func(context.Context) error:
		return ActionFunc(func(c *Context) error {
			return a(c.Context)
		})
	case func(context.Context):
		return ActionFunc(func(c *Context) error {
			a(c.Context)
			return nil
		})
	case func() error:
		return ActionFunc(func(*Context) error {
			return a()
		})
	case func():
		return ActionFunc(func(*Context) error {
			a()
			return nil
		})
	case func(*Context, Action) error:
		return middlewareFunc(a)
	case func(Action) Action:
		return middlewareFunc(func(c *Context, next Action) error {
			return c.Do(a(next))
		})
	}
	panic(fmt.Sprintf("unexpected type: %T", item))
}

// ContextValue provides an action which updates the context with a
// value.
func ContextValue(key, value interface{}) Action {
	return Before(ActionFunc(func(c *Context) error {
		c.Context = context.WithValue(c.Context, key, value)
		return nil
	}))
}

// SetContext provides an action which sets the context
func SetContext(ctx context.Context) Action {
	return ActionFunc(func(c *Context) error {
		c.Context = ctx
		return nil
	})
}

// Timeout provides an action which adds a timeout to the context.
func Timeout(timeout time.Duration) Action {
	return ActionFunc(func(c *Context) error {
		return c.Before(func(c1 *Context) error {
			ctx, cancel := context.WithTimeout(c1.Context, timeout)
			return c1.Do(
				SetContext(ctx),
				After(ActionFunc(func(*Context) error {
					cancel()
					return nil
				})),
			)
		})
	})
}

// SetValue provides an action which sets the value of the flag or argument.
func SetValue(v string) Action {
	return ActionFunc(func(c *Context) error {
		return c.SetValue(v)
	})
}

// Bind invokes a function, either using the value specified or the value read from
// the flag or arg.  The bind function is the function to execute, and valopt is optional,
// and if specified, indicates the value to set; otherwise, the value is read from the flag.
func Bind[V any](bind func(V) error, valopt ...V) Action {
	proto := &Prototype{Value: bindSupportedValue(new(V))}
	switch len(valopt) {
	case 0:
		return Pipeline(proto, bindTiming(ActionFunc(func(c *Context) error {
			if !c.Seen("") {
				return nil
			}
			return bind(c.Value("").(V))
		})))

	case 1:
		val := valopt[0]
		return Pipeline(proto, bindTiming(ActionFunc(func(c *Context) error {
			if !c.Seen("") {
				return nil
			}
			return bind(val)
		})))
	default:
		panic("expected 0 or 1 args for valopt")
	}
}

// BindContext binds a value to a context value.
// The value function determines how to obtain the value from the context.  Usually, this
// is a call to context/Context.Value.   The bind function is the function to set the value,
// and valopt is optional, and if specified, indicates the value to set; otherwise, the
// value is read from the flag.
func BindContext[T, V any](value func(context.Context) *T, bind func(*T, V) error, valopt ...V) Action {
	return bindThunk(func(c *Context) *T {
		return value(c)
	}, bind, valopt...)
}

// BindIndirect binds a value to the specified option indirectly.
// For example, it is common to define a FileSet arg and a Boolean flag that
// controls whether or not the file set is enumerated recursively.  You can use
// BindIndirect to update the arg indirectly by naming it and the bind function:
//
//    &cli.Arg{
//        Name: "files",
//        Value: new(cli.FileSet),
//    }
//    &cli.Flag{
//        Name: "recursive",
//        HelpText: "Whether files is recursively searched",
//        Action: cli.BindIndirect("files", (*cli.FileSet).SetRecursive),
//    }
//
// The name parameter specifies the name of the flag or arg that is affected.  The
// bind function is the function to set the value, and valopt is optional, and if specified,
// indicates the value to set; otherwise, the value is read from the flag.
func BindIndirect[T, V any](name string, bind func(*T, V) error, valopt ...V) Action {
	return bindThunk(func(c *Context) *T {
		return c.Value(name).(*T)
	}, bind, valopt...)
}

func bindThunk[T, V any](thunk func(*Context) *T, bind func(*T, V) error, valopt ...V) Action {
	proto := &Prototype{Value: bindSupportedValue(new(V))}
	switch len(valopt) {
	case 0:
		return Pipeline(proto, bindTiming(ActionFunc(func(c *Context) error {
			if !c.Seen("") {
				return nil
			}
			return bind(thunk(c), c.Value("").(V))
		})))

	case 1:
		val := valopt[0]
		return Pipeline(proto, bindTiming(ActionFunc(func(c *Context) error {
			if !c.Seen("") {
				return nil
			}
			return bind(thunk(c), val)
		})))
	default:
		panic("expected 0 or 1 args for valopt")
	}
}

func bindTiming(a Action) Action {
	return AtTiming(a, ActionTiming)
}

func bindSupportedValue(v interface{}) interface{} {
	// Bind functions will either use *V or V depending upon what
	// supports the built-in convention values or implements Value.
	// Any built-in primitive will work as is.  However, if v is actually
	// *V but V is **W and W is a Value implementation, then unwrap this
	// so we end up with *W.  For example, instead of **FileSet, just use *FileSet.
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Ptr {
		pointsToValue := val.Elem().Type()
		if pointsToValue.Implements(valueType) {
			return reflect.New(pointsToValue.Elem()).Interface()
		}
	}

	// Primitives and other values
	return v
}

// Accessory provides an action which sets up an accessory flag for the current flag or argument.
// A common pattern is that a flag has a related sibling flag that can be used to refine the value.
// For example, you might define a --recursive flag next to a FileSet argument.  When a Value
// supports an accessory flag prototype, you can use this action to activate it from its Uses pipeline.
func Accessory[T Value](name string, fn func(T) Prototype) Action {
	return ActionFunc(func(c *Context) error {
		val := c.Value("").(T)
		proto := fn(val)

		if proto.Category == "" {
			proto.Category = c.option().category()
		}
		proto.HelpText = conditionalFormat(proto.HelpText, c.Name())
		proto.ManualText = conditionalFormat(proto.ManualText, c.Name())
		proto.Description = conditionalFormat(proto.Description, c.Name())

		switch name {
		case "":
			// user specified value
		case "-":
			proto.Name = withoutDecorators(c.Name()) + "-" + proto.Name
		default:
			proto.Name = name
		}

		f := &Flag{
			Uses: proto,
		}
		return c.Do(AddFlag(f))
	})
}

func conditionalFormat(s string, args ...interface{}) string {
	if strings.Contains(s, "%") {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// Enum provides validation that a particular flag or arg value matches a given set of
// legal values.  The operation applies to the raw values in the occurrences.
// When used, it also sets the flag synopsis to a reasonable default derived from the values
// unless the flag provides its own specific synopsis.  This enables completion on the enumerated
// values.
func Enum(options ...string) Action {
	oset := map[string]bool{}
	for _, o := range options {
		oset[o] = true
	}
	var usageText string
	if len(options) > 3 {
		usageText = "(" + strings.Join(options[0:3], "|") + "|...)"
	} else {
		usageText = "(" + strings.Join(options, "|") + ")"
	}

	return &Prototype{
		UsageText:  usageText,
		Completion: CompletionValues(options...),
		Setup: Setup{
			Uses: ValidatorFunc(func(raw []string) error {
				for _, occur := range raw {
					if _, ok := oset[occur]; !ok {
						expected := listOfValues(options)
						return &ParseError{
							Code: InvalidArgument,
							Err: errorTemplate{
								fallback: fmt.Sprintf("unrecognized value %q, expected %s", occur, expected),
								format:   fmt.Sprintf("unrecognized value %q for %%[1]s, expected %s", occur, expected),
							},
						}
					}
				}
				return nil
			}),
		},
	}
}

// Data sets metadata for a command, flag, arg, or expression.  This handler is generally
// set up inside a Uses pipeline.
// When value is nil, the corresponding
// metadata is deleted
func Data(name string, value interface{}) Action {
	return ActionFunc(func(c *Context) error {
		c.SetData(name, value)
		return nil
	})
}

// Category sets the category of a command, flag, or expression.  This handler is generally
// set up inside a Uses pipeline.
func Category(name string) Action {
	return ActionFunc(func(c *Context) error {
		c.target().setCategory(name)
		return nil
	})
}

// HookBefore registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookBefore(pattern string, handler Action) Action {
	return ActionFunc(func(c *Context) error {
		return c.HookBefore(pattern, handler)
	})
}

// HookAfter registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookAfter(pattern string, handler Action) Action {
	return ActionFunc(func(c *Context) error {
		return c.HookAfter(pattern, handler)
	})
}

// HandleSignal provides an action that provides simple handling of a signal, usually os.Interrupt.
// HandleSignal updates the Context to handle the signal by exposing the context Done() channel.
// Compare this behavior to os/signal.NotifyContext.  Here's an example:
//
//       &cli.Command{
//          Name: "command",
//          Uses: cli.HandleSignal(os.Interrupt),
//          Action: func(c *cli.Context) error {
//              for {
//                  select {
//                  case <-c.Done():
//                      // Ctrl+C was called
//                      return nil
//                  default:
//                      // process another step, use return to exit
//                  }
//              }
//          }
//       }
//
// The signal handler is unregistered in the After pipeline.  The recommended approach
// is therefore to place cleanup into After and consider using a timeout.
// The process will be terminated when the user presses ^C for the second time:
//
func HandleSignal(s os.Signal) Action {
	return ActionFunc(func(c *Context) error {
		return c.Before(func(c1 *Context) error {
			ctx, stop := signal.NotifyContext(c1.Context, s)
			return c1.Do(
				SetContext(ctx),
				After(ActionFunc(func(*Context) error {
					stop()
					return nil
				})),
			)
		})
	})
}

// OptionalValue makes the flag's value optional, and when its value is not specified, the implied value
// is set to this value v.  Say that a flag is defined as:
//
//   &Flag {
//     Name: "secure",
//     Value: cli.String(),
//     Uses: cli.Optional("TLS1.2"),
//   }
//
// This example implies that --secure without a value is set to the value TLS1.2 (presumably other versions
// are allowed).  This example is a fair use case of this feature: making a flag opt-in to some sort of default
// configuration and allowing an expert configuration by using a value.
// For short options, no space can be between the flag and value (e.g. you need -sString to
// specify a String to the -s option).
// In general, making the value of a non-Boolean flag optional is not recommended when
// the command also allows arguments because it can make the syntax ambiguous.
func OptionalValue(v interface{}) Action {
	return Initializer(ActionFunc(func(c *Context) error {
		c.Flag().setOptionalValue(v)
		return nil
	}))
}

// ProvideValueInitializer causes an additional child context to be created
// which is used to initialize an arbitrary value.  For more information,
// refer to the implementation provided by Context.ProvideValueInitializer.
func ProvideValueInitializer(v target, name string, a Action) Action {
	return ActionFunc(func(c *Context) error {
		return c.ProvideValueInitializer(v, name, a)
	})
}

// AddFlag provides an action which adds a flag to the command or app
func AddFlag(f *Flag) Action {
	return ActionFunc(func(c *Context) error {
		return c.AddFlag(f)
	})
}

// AddCommand provides an action which adds a sub-command to the command or app
func AddCommand(v *Command) Action {
	return ActionFunc(func(c *Context) error {
		return c.AddCommand(v)
	})
}

// AddArg provides an action which adds an arg to the command or app
func AddArg(a *Arg) Action {
	return ActionFunc(func(c *Context) error {
		return c.AddArg(a)
	})
}

// RemoveArg provides an action which removes an arg from the command or app.
// The name specifies the name, index, or arg itself
func RemoveArg(name interface{}) Action {
	return ActionFunc(func(c *Context) error {
		return c.RemoveArg(name)
	})
}

// AddFlags provides an action which adds the specified flags to the command
func AddFlags(flags ...*Flag) Action {
	return ActionFunc(func(c *Context) error {
		return c.AddFlags(flags...)
	})
}

// AddArgs provides an action which adds the specified args to the command
func AddArgs(args ...*Arg) Action {
	return ActionFunc(func(c *Context) error {
		return c.AddArgs(args...)
	})
}

// AddCommands provides an action which adds the specified commands to the command
func AddCommands(commands ...*Command) Action {
	return ActionFunc(func(c *Context) error {
		return c.AddCommands(commands...)
	})
}

// FlagSetup action is used to apply set-up to a flag.  This is typically
// used within the Uses pipeline for actions that provide some default
// setup.  The setup function fn will only be called in the initialization
// timing and only if setup hasn't been blocked by PreventSetup.
func FlagSetup(fn func(*Flag)) Action {
	return optionalSetup(func(c *Context) {
		if f := c.Flag(); f != nil {
			fn(f)
		}
	})
}

// ArgSetup action is used to apply set-up to a Arg.  This is typically
// used within the Uses pipeline for actions that provide some default
// setup.  The setup function fn will only be called in the initialization
// timing and only if setup hasn't been blocked by PreventSetup.
func ArgSetup(fn func(*Arg)) Action {
	return optionalSetup(func(c *Context) {
		if a := c.Arg(); a != nil {
			fn(a)
		}
	})
}

// CommandSetup action is used to apply set-up to a Command.  This is typically
// used within the Uses pipeline for actions that provide some default
// setup.  The setup function fn will only be called in the initialization
// timing and only if setup hasn't been blocked by PreventSetup.
func CommandSetup(fn func(*Command)) Action {
	return optionalSetup(func(c *Context) {
		if cmd := c.Command(); cmd != nil {
			fn(cmd)
		}
	})
}

func optionalSetup(a func(*Context)) ActionFunc {
	return func(c *Context) error {
		if c.SkipImplicitSetup() {
			return nil
		}
		if c.IsInitializing() {
			a(c)
		}
		return nil
	}
}

// ImplicitValue sets the implicit value which is specified for the arg or flag
// if it was not specified for the command.  Any errors are suppressed
func ImplicitValue(fn func(*Context) (string, bool)) Action {
	return AtTiming(ActionFunc(func(c *Context) error {
		if c.Occurrences("") == 0 {
			if v, ok := fn(c); ok {
				c.SetValue(v)
			}
		}
		return nil
	}), ImplicitValueTiming)
}

// Implies is used to set the implied value of another flag.   For example,
// an app might have two flags, --mode and --encryption-key, and you might allow --encryption-key
// to imply --mode=encrypt which saves power users from having to type both.  Because it is an
// implied value, if the other flag is explicitly specified, the explicit value wins regardless of
// its position in the command line.  If the name is the empty string, returns a no-op.
func Implies(name, value string) Action {
	if name == "" {
		return nil
	}
	if name[0] != '-' {
		name = "--" + name
	}
	return ActionFunc(func(c *Context) error {
		return c.Parent().HookBefore(name, ImplicitValue(func(_ *Context) (string, bool) {
			return value, true
		}))
	})
}

// Customize matches a flag, arg, or command and runs additional pipeline steps.  Customize
// is usually used to apply further customizations after an extension has done setup of
// the defaults.
func Customize(pattern string, a ...Action) Action {
	return ActionFunc(func(c *Context) error {
		return c.Customize(pattern, a...)
	})
}

// Transform defines how to interpret raw values passed to a flag or arg.  The action
// is added to the Uses pipeline.  The typical use of transforms is to interpret the
// value passed to an argument as instead a reference to a file which is loaded.
// The function fn can return string, []byte, or io.Reader.  If the Value implements
// method SetData(io.Reader) error, then this is called instead when setting the Value.
// If it doesn't, then bytes or readers are read in and treated as a string and the
// usual Set method is used.  See also: FileReference and AllowFileReference, which provide
// common transforms.
func Transform(fn func([]string) (interface{}, error)) Action {
	return ActionFunc(func(c *Context) error {
		c.option().setTransform(fn)
		return nil
	})
}

func (p Prototype) Execute(c *Context) error {
	return c.Do(FlagSetup(p.copyToFlag), ArgSetup(p.copyToArg), p.Setup)
}

func (p *Prototype) copyToArg(o *Arg) {
	if o.Name == "" {
		o.Name = p.Name
	}
	if o.Category == "" {
		o.Category = p.Category
	}
	if o.HelpText == "" {
		o.HelpText = p.HelpText
	}
	if o.ManualText == "" {
		o.ManualText = p.ManualText
	}
	if o.UsageText == "" {
		o.UsageText = p.UsageText
	}
	if o.Description == "" {
		o.Description = p.Description
	}
	if o.FilePath == "" {
		o.FilePath = p.FilePath
	}
	if o.DefaultText == "" {
		o.DefaultText = p.DefaultText
	}
	if o.Completion == nil {
		o.Completion = p.Completion
	}
	if p.Value != nil && (o.option.flags.destinationImplicitlyCreated() || o.Value == nil) {
		o.Value = p.Value
		o.option.flags = o.option.flags & ^internalFlagDestinationImplicitlyCreated
	}

	o.EnvVars = append(o.EnvVars, p.EnvVars...)
	o.Options |= p.Options
	update(o.Data, p.Data)
}

func (p *Prototype) copyToFlag(o *Flag) {
	if o.Name == "" {
		o.Name = p.Name
	}
	if o.Category == "" {
		o.Category = p.Category
	}
	if o.HelpText == "" {
		o.HelpText = p.HelpText
	}
	if o.ManualText == "" {
		o.ManualText = p.ManualText
	}
	if o.UsageText == "" {
		o.UsageText = p.UsageText
	}
	if o.Description == "" {
		o.Description = p.Description
	}
	if o.FilePath == "" {
		o.FilePath = p.FilePath
	}
	if o.DefaultText == "" {
		o.DefaultText = p.DefaultText
	}
	if o.Completion == nil {
		o.Completion = p.Completion
	}
	if p.Value != nil && (o.option.flags.destinationImplicitlyCreated() || o.Value == nil) {
		o.Value = p.Value
		o.option.flags = o.option.flags & ^internalFlagDestinationImplicitlyCreated
	}

	o.Aliases = append(o.Aliases, p.Aliases...)
	o.EnvVars = append(o.EnvVars, p.EnvVars...)
	o.Options |= p.Options
	update(o.Data, p.Data)
}

// Execute the action by calling the function
func (af ActionFunc) Execute(c *Context) error {
	if af == nil {
		return nil
	}
	return af(c)
}

// Append appends an action to the pipeline
func (p ActionPipeline) Append(x ...Action) ActionPipeline {
	return ActionPipeline(append(p, x...))
}

// Execute the pipeline by calling each action successively
func (p ActionPipeline) Execute(c *Context) (err error) {
	if p == nil {
		return nil
	}
	return p.toCons().Execute(c)
}

func (p ActionPipeline) flatten() []Action {
	if !p.anyNested() {
		return p
	}

	result := make([]Action, 0, len(p))
	for _, a := range p {
		if n, ok := a.(ActionPipeline); ok {
			result = append(result, n.flatten()...)
			continue
		}
		result = append(result, a)
	}
	return result
}

func (p ActionPipeline) anyNested() bool {
	for _, i := range p {
		if _, ok := i.(ActionPipeline); ok {
			return true
		}
	}
	return false
}

func (p ActionPipeline) toCons() *cons {
	var head *cons
	for i := len(p) - 1; i >= 0; i-- {
		head = &cons{
			action: p[i],
			next:   head,
		}
	}
	return head
}

func (o *cons) Execute(c *Context) error {
	if o == nil {
		return nil
	}
	if m, ok := o.action.(Middleware); ok {
		return m.ExecuteWithNext(c, o.next)
	}

	if err := execute(c, o.action); err != nil {
		return err
	}
	return execute(c, o.next)
}

func (p *actionPipelines) add(t Timing, h Action) {
	switch t {
	case InitialTiming:
		p.Initializers = Pipeline(p.Initializers, h)

	case BeforeTiming, ValidatorTiming, ImplicitValueTiming, justBeforeTiming:
		actualIndex := actualBeforeIndex[t]
		p.Before[actualIndex] = Pipeline(p.Before[actualIndex], h)

	case ActionTiming:
		// As a rule, middleware wraps the existing Action pipeline.
		// This solves for when middleware was added in the Uses or Before
		// pipelines.  This may place middleware in the wrong nesting but
		// hopefully most middleware is designed to work without ordering
		if _, ok := h.(Middleware); ok {
			p.Action = Pipeline(h, p.Action)
		} else {
			p.Action = Pipeline(p.Action, h)
		}
	case AfterTiming:
		p.After = Pipeline(p.After, h)
	default:
		panic("unreachable!")
	}
}

func (w withTimingWrapper) Execute(c *Context) error {
	switch w.t {
	case ImplicitValueTiming:
		return c.act(Pipeline(
			Data(implicitTimingEnabledKey, true),
			w.Action,
			Data(implicitTimingEnabledKey, nil),
		), BeforeTiming, false)
	}

	return c.act(w.Action, w.t, false)
}

func (b beforePipeline) Execute(c *Context) error {
	return ActionPipeline(append(append(append(b[0], b[1]...), b[2]...), b[3]...)).Execute(c)
}

func (i *hooksSupport) hookBefore(pat string, a Action) error {
	i.before = append(i.before, &hook{newContextPathPattern(pat), a})
	return nil
}

func (i *hooksSupport) executeBeforeHooks(target *Context) error {
	for _, b := range i.before {

		if b.pat.Match(target.Path()) {
			err := target.Do(b.action)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *hooksSupport) hookAfter(pat string, a Action) error {
	i.after = append(i.after, &hook{newContextPathPattern(pat), a})
	return nil
}

func (i *hooksSupport) executeAfterHooks(target *Context) error {
	for _, b := range i.after {
		if b.pat.Match(target.Path()) {
			err := target.Do(b.action)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *customizableSupport) customize(pat string, handler Action) {
	i.items = append(i.items, &hook{newContextPathPattern(pat), handler})
}

func (i *customizableSupport) customizations() []*hook {
	return i.items
}

func (s *pipelinesSupport) uses() *actionPipelines {
	return s.p
}

func (s *pipelinesSupport) setPipelines(p *actionPipelines) {
	s.p = p
}

func (s *pipelinesSupport) appendAction(t Timing, ah Action) {
	s.p.add(t, ah)
}

func (m middlewareFunc) Execute(c *Context) error {
	return m.ExecuteWithNext(c, nil)
}

func (m middlewareFunc) ExecuteWithNext(c *Context, a Action) error {
	return m(c, a)
}

func (v ValidatorFunc) Execute(c *Context) error {
	return c.Do(AtTiming(ActionFunc(func(c *Context) error {
		occur := c.RawOccurrences("")
		if err := v(occur); err != nil {
			return argTakerError(c.Name(), "", err, nil)
		}

		return nil
	}), ValidatorTiming))
}

func emptyActionImpl(*Context) error {
	return nil
}

func execute(c *Context, af Action) error {
	if af == nil {
		return nil
	}
	return af.Execute(c)
}

func doThenExit(a Action) Action {
	return ActionFunc(func(c *Context) error {
		err := a.Execute(c)
		if err != nil {
			return err
		}
		return Exit(0)
	})
}

func newPipelines(uses Action, opts *Option) *actionPipelines {
	// PreventSetup if specified must be handled first
	first := *opts & PreventSetup

	// Use a reference to the options so that if it is updated, the
	// most recent version will apply when the pipeline actually runs
	return &actionPipelines{
		Initializers: Pipeline(first, uses, opts),
	}
}

var (
	_ Action   = withTimingWrapper{}
	_ Action   = Setup{}
	_ Action   = Prototype{}
	_ Action   = beforePipeline{}
	_ Action   = (*cons)(nil)
	_ hookable = (*hooksSupport)(nil)
)
