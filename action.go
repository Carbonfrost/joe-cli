package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/signal"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli/internal/support"
)

// ActionFunc provides the basic function for an Action
type ActionFunc func(*Context) error

type actionFunc func(context.Context) error

//counterfeiter:generate . Action

// Action represents the building block of the various actions
// to perform when an app, command, flag, or argument is being evaluated.
type Action interface {

	// Execute will execute the action.  If the action returns an error, this
	// may cause subsequent actions in the pipeline not to be run and cause
	// the app to exit with an error exit status.
	Execute(context.Context) error
}

// legacyAction uses *Context instead of context.Context in the Execute func.
// Allow this to be used in ActionOf
type legacyAction interface {
	Execute(*Context) error
}

type legacyActionWrapper struct {
	a legacyAction
}

//counterfeiter:generate . Middleware

// Middleware provides an action which controls how and whether the next
// action in the pipeline is executed
type Middleware interface {
	Action

	// ExecuteWithNext will execute the action and invoke Execute on the next
	// action
	ExecuteWithNext(context.Context, Action) error
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

// Prototype implements an action which sets up a flag, arg, or command.  The
// prototype copies its values to the corresponding target if they have not
// already been set.  Irrelevant fields are not set and do not cause errors; for example,
// setting FilePath, Value, and EnvVars, for a Command prototype has no effect.
// Some values are merged rather than overwritten:
// Data, Options, EnvVars, and Aliases.
// If setup has been prevented with the PreventSetup action,
// the prototype will do nothing.  The main use of prototype is in extensions to provide
// reasonable defaults
type Prototype struct {
	Aliases     []string
	Category    string
	Data        map[string]interface{}
	DefaultText string
	Description interface{}
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
	NArg        interface{}
}

// ValidatorFunc defines an Action that applies a validation rule to
// the explicit raw occurrence values for a flag or argument.
type ValidatorFunc func(s []string) error

// TransformFunc implements a transformation from raw occurrences, which customizes
// the behavior of parsing. The function can return string, []byte, or io.Reader.
type TransformFunc func(rawOccurrences []string) (interface{}, error)

type hookable interface {
	hook(at Timing, handler Action) error
	executeInitializeHooks(context.Context) error
	executeBeforeHooks(context.Context) error
	executeAfterHooks(context.Context) error
	valueTargets() []*valueTarget
	addValueTarget(*valueTarget)
}

type target interface {
	SetHidden(bool)
	SetData(name string, v interface{})
	LookupData(name string) (interface{}, bool)

	uses() *actionPipelines
	options() *Option
	pipeline(Timing) interface{}
	appendAction(Timing, Action)
	setDescription(any)
	setHelpText(string)
	setUsageText(string)
	setManualText(string)
	setCategory(name string)
	setCompletion(Completion)
	description() any
	helpText() string
	usageText() string
	manualText() string
	category() string
	setInternalFlags(internalFlags, bool)
	internalFlags() internalFlags
	completion() Completion
	data() map[string]any
}

type hooksSupport struct {
	hooks  actionPipelines
	values []*valueTarget
}

type pipelinesSupport struct {
	p actionPipelines
}

type internalFlagsSupport struct {
	flags internalFlags
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
type beforePipeline [actualBeforeIndexMaxValue]ActionPipeline

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
	// for an arg or flag.  This timing can be set with the At function which affects
	// the sort order of actions so that validation occurs before all other actions in Before
	// pipeline.  When the action runs, the actual timing will be BeforeTiming.
	ValidatorTiming

	// ImplicitValueTiming represents timing that happens when an implied value is being
	// computed for an arg or flag.  This timing can be set with the At function
	// which affects the sort order of actions so that implied value timing occurs just before
	// the action.  When the action runs, the actual timing will be BeforeTiming.
	ImplicitValueTiming

	// justBeforeTiming is internally used for actions that must happen just before the
	// Action timing
	justBeforeTiming
)

const (
	panicDataKey = "__PanicData"
)

const (
	actualBeforeIndexValidatorTiming = iota
	actualBeforeIndexBeforeTiming
	actualBeforeIndexImplicitValueTiming
	actualBeforeIndexJustBeforeTiming
	actualBeforeIndexMaxValue
)

var (
	emptyAction Action = ActionFunc(emptyActionImpl)
	patFlagName        = regexp.MustCompile(`{}`)

	actualBeforeIndex = map[Timing]int{
		ValidatorTiming:     actualBeforeIndexValidatorTiming,
		BeforeTiming:        actualBeforeIndexBeforeTiming,
		ImplicitValueTiming: actualBeforeIndexImplicitValueTiming,
		justBeforeTiming:    actualBeforeIndexJustBeforeTiming,
	}

	triggerBeforeOptions = triggerOptionsHO(BeforeTiming, (*Context).executeBefore)
	triggerAfterOptions  = triggerOptionsHO(AfterTiming, (*Context).executeAfter)

	triggerBeforeValueTargets = triggerValueTargetsHO(BeforeTiming, (*Context).executeBefore)
	triggerAfterValueTargets  = triggerValueTargetsHO(AfterTiming, (*Context).executeAfter)
	triggerValueTargets       = triggerValueTargetsHO(ActionTiming, (*Context).executeSelf)
	initializeValueTargets    = triggerValueTargetsHO(InitialTiming, (*Context).initialize)

	// defaultCommand defines the flow for how a command is executed during the
	// initialization and other pipeline stages.  (See also defaultOption.)
	//
	// _Deferred pipelines_ accumulate actions that were registered in previous
	//    steps in the flow, including possibly from other targets.  Note that
	//    executeDeferredPipeline occurs BEFORE user pipelines because the user
	//    cannot re-entrantly queue additional deferrals at the particular timing
	//    anyway.  (Said another way and by example, calling Before() within a
	//    Before action just invokes it immediately rather than try queueing it.)
	//
	defaultCommand = actionPipelines{
		Initializers: actions(
			IfMatch(RootCommand,
				actions(
					ActionFunc(setupDefaultData),
					ActionFunc(setupDefaultIO),
					ActionFunc(fixupCommandInternals),
					ActionFunc(setupDefaultTemplateFuncs),
					ActionFunc(setupDefaultTemplates),
				),
			),
			actionFunc(preventSetupIfPresent),
			executeDeferredPipeline(InitialTiming),
			executeUserPipeline(InitialTiming),
			actionFunc(applyUserOptions),
			ActionFunc(applyImplicitVisibility),
			IfMatch(RootCommand,
				actions(
					ActionFunc(optionalCommand("help", defaultHelpCommand)),
					ActionFunc(optionalFlag("help", defaultHelpFlag)),
					ActionFunc(optionalCommand("version", defaultVersionCommand)),
					ActionFunc(optionalFlag("version", defaultVersionFlag)),
					actionFunc(setupCompletion),
				),
			),
			actionFunc(ensureSubcommands),
			ActionFunc(initializeFlagsArgs),
			ActionFunc(initializeSubcommands),
			ActionFunc(initializeValueTargets),
			ActionFunc(copyContextToParent),
		),
		Action: actions(
			ActionFunc(triggerOptions),
			ActionFunc(triggerValueTargets),
			IfMatch(subcommandDidNotExecute,
				actions(
					executeDeferredPipeline(ActionTiming),
					actionFunc(triggerRobustParsingAndCompletion),
					executeUserPipeline(ActionTiming),
				),
			),
		),
		Before: beforePipeline{
			nil,
			actions(
				executeDeferredPipeline(BeforeTiming),
				executeUserPipeline(BeforeTiming),
				ActionFunc(triggerBeforeOptions),
				ActionFunc(triggerBeforeValueTargets),
			),
			nil,
		},
		After: actions(
			executeDeferredPipeline(AfterTiming),
			executeUserPipeline(AfterTiming),
			ActionFunc(triggerAfterOptions),
			ActionFunc(triggerAfterValueTargets),
			ActionFunc(failWithContextError),
		),
	}

	defaultOption = actionPipelines{
		Initializers: actions(
			actionFunc(preventSetupIfPresent),
			executeDeferredPipeline(InitialTiming),
			executeUserPipeline(InitialTiming),
			actionFunc(applyUserOptions),
			ActionFunc(applyImplicitVisibility),
			actionFunc(setupValueInitializer),
			actionFunc(setupOptionFromEnv),
			ActionFunc(initializeValueTargets),
			ActionFunc(checkForSupportedFlagType),
			ActionFunc(setInternalFlag(internalFlagInitialized)),
		),
		Before: beforePipeline{
			actualBeforeIndexBeforeTiming: actions(
				executeDeferredPipeline(BeforeTiming),
				executeUserPipeline(BeforeTiming),
				ActionFunc(triggerBeforeValueTargets),
			),
			actualBeforeIndexJustBeforeTiming: actions(
				ActionFunc(checkForRequiredOption),
			),
		},
		Action: actions(
			actionFunc(executeOptionPipeline),
			ActionFunc(triggerValueTargets),
		),
		After: actions(
			executeDeferredPipeline(AfterTiming),
			executeUserPipeline(AfterTiming),
			ActionFunc(triggerAfterValueTargets),
		),
	}

	defaultValue = actionPipelines{
		Initializers: actions(
			executeDeferredPipeline(InitialTiming),
			executeUserPipeline(InitialTiming),
			ActionFunc(initializeFlagsArgs),
		),
		Before: beforePipeline{
			nil,
			actions(
				executeDeferredPipeline(BeforeTiming),
				executeUserPipeline(BeforeTiming),
				ActionFunc(triggerBeforeOptions),
			),
			nil,
		},
		Action: actions(
			ActionFunc(triggerOptions),
			executeDeferredPipeline(ActionTiming),
			executeUserPipeline(ActionTiming),
		),
		After: actions(
			executeDeferredPipeline(AfterTiming),
			executeUserPipeline(AfterTiming),
			ActionFunc(triggerAfterOptions),
		),
	}

	errCantHook     = errors.New("hooks are not supported in this context")
	errAssertFailed = errors.New("context does not meet requirements for action")

	timingLabels = map[Timing][2]string{
		ActionTiming:        {"action timing", "ACTION"},
		AfterTiming:         {"after timing", "AFTER"},
		BeforeTiming:        {"before timing", "BEFORE"},
		ImplicitValueTiming: {"implicit value timing", "IMPLICIT_VALUE"},
		InitialTiming:       {"initial timing", "INITIAL"},
		ValidatorTiming:     {"validator timing", "VALIDATOR"},
	}

	// ErrImplicitValueAlreadySet occurs when the value for a flag
	// or arg is set multiple times within the implicit timing.
	ErrImplicitValueAlreadySet = errors.New("value already set implicitly")
)

// Execute executes the Setup, which assigns the various parts to their
// pipelines
func (s Setup) Execute(ctx context.Context) error {
	c := FromContext(ctx)
	if s.Uses != nil {
		if err := c.act(s.Uses, InitialTiming, s.Optional); err != nil {
			return err
		}
	}
	if s.Before != nil {
		if err := c.Before(ActionOf(s.Before)); err != nil {
			return err
		}
	}
	if err := c.Action(ActionOf(s.Action)); err != nil {
		return err
	}
	if err := c.After(ActionOf(s.After)); err != nil {
		return err
	}
	return nil
}

func (s Setup) Use(action Action) Setup {
	s.Uses = Pipeline(s.Uses).Append(action)
	return s
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

// Do invokes the actions with the given context
func Do(c context.Context, action Action) error {
	if action == nil {
		return nil
	}
	return action.Execute(c)
}

// SuppressError wraps an action to ignore its error.
func SuppressError(a Action) Action {
	return ActionOf(func(c context.Context) error {
		a.Execute(c)
		return nil
	})
}

// Recover wraps an action to recover from a panic
func Recover(a Action) Action {
	return actionFunc(func(c context.Context) error {
		defer func() {
			if rvr := recover(); rvr != nil {
				FromContext(c).SetData(panicDataKey, &panicData{
					recovered: fmt.Sprint(rvr),
					stack:     formatStack(),
				})
			}
		}()
		return Do(c, a)
	})
}

func formatStack() string {
	return string(debug.Stack())
}

func failWithContextError(c *Context) error {
	if r, ok := c.LookupData(panicDataKey); ok {
		rvr := r.(*panicData)
		fmt.Fprint(c.Stderr, rvr.stack)
		return c.internalError(fmt.Errorf(rvr.recovered))
	}
	return nil
}

// Before revises the timing of the action so that it runs in the Before pipeline.
// This function is used to wrap actions in the initialization pipeline that will be
// deferred until later.
func Before(a Action) Action {
	return At(BeforeTiming, a)
}

// After revises the timing of the action so that it runs in the After pipeline.
// This function is used to wrap actions in the initialization pipeline that will be
// deferred until later.
func After(a Action) Action {
	return At(AfterTiming, a)
}

// Initializer marks an action handler as being for the initialization phase.  When such a handler
// is added to the Uses pipeline, it will automatically be associated correctly with the initialization
// of the value.  Otherwise, this handler is not special
func Initializer(a Action) Action {
	return At(InitialTiming, a)
}

// At wraps an action and causes it to execute at the given timing.
func At(t Timing, a Action) Action {
	return withTimingWrapper{a, t}
}

// ActionOf converts a value to an Action.  Any of the following types can be converted:
//
//   - func(*Context) error
//   - func(*Context)
//   - func(context.Context) error  (same signature as Action.Execute)
//   - func(context.Context)
//   - func() error
//   - func()
//   - Action
//
// You can also implement the legacy Action interface which used *Context instead of
// context.Context in the Execute method (i.e. Execute(*Context) error)
// As a special case, these signatures are allowed in order to provide middleware:
//
//   - func(Action)Action
//   - func(*Context, Action) error
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
	case legacyAction:
		return legacyActionWrapper{a}
	case func(*Context):
		return ActionFunc(func(c *Context) error {
			a(c)
			return nil
		})
	case func(context.Context) error:
		return actionFunc(a)
	case func(context.Context):
		return actionFunc(func(c context.Context) error {
			a(c)
			return nil
		})
	case func() error:
		return actionFunc(func(context.Context) error {
			return a()
		})
	case func():
		return actionFunc(func(context.Context) error {
			a()
			return nil
		})
	case func(*Context, Action) error:
		return middlewareFunc(a)
	case func(Action) Action:
		return middlewareFunc(func(c *Context, next Action) error {
			return Do(c, a(next))
		})
	}
	panic(fmt.Sprintf("unexpected type: %T", item))
}

// ContextValue provides an action which updates the context with a
// value.
func ContextValue(key, value any) Action {
	return actionThunk2((*Context).SetContextValue, key, value)
}

// SetContext provides an action which sets the context
func SetContext(ctx context.Context) Action {
	return actionThunk1((*Context).SetContext, ctx)
}

// Timeout provides an action which adds a timeout to the context.
func Timeout(timeout time.Duration) Action {
	return Before(actionFunc(func(c context.Context) error {
		ctx, cancel := context.WithTimeout(c, timeout)
		return Do(c, Pipeline(
			SetContext(ctx),
			After(ActionOf((func())(cancel))),
		))
	}))
}

// SetValue provides an action which sets the value of the flag or argument.
func SetValue(v any) Action {
	return actionThunk1((*Context).SetValue, v)
}

// Bind invokes a function, either using the value specified or the value read from
// the flag or arg.  The bind function is the function to execute, and valopt is optional,
// and if specified, indicates the value to set; otherwise, the value is read from the flag.
func Bind[V any](bind func(V) error, valopt ...V) Action {
	return bindThunk(FromContext, func(_ *Context, v V) error {
		return bind(v)
	}, valopt...)
}

// BindContext binds a value to a context value.
// The value function determines how to obtain the value from the context.  Usually, this
// is a call to context/Context.Value.   The bind function is the function to set the value,
// and valopt is optional, and if specified, indicates the value to set; otherwise, the
// value is read from the flag.
func BindContext[T, V any](value func(context.Context) *T, bind func(*T, V) error, valopt ...V) Action {
	return bindThunk(func(c context.Context) *T {
		return value(c)
	}, bind, valopt...)
}

// BindIndirect binds a value to the specified option indirectly.
// For example, it is common to define a FileSet arg and a Boolean flag that
// controls whether or not the file set is enumerated recursively.  You can use
// BindIndirect to update the arg indirectly by naming it and the bind function:
//
//	&cli.Arg{
//	    Name: "files",
//	    Value: new(cli.FileSet),
//	}
//	&cli.Flag{
//	    Name: "recursive",
//	    HelpText: "Whether files is recursively searched",
//	    Action: cli.BindIndirect("files", (*cli.FileSet).SetRecursive),
//	}
//
// The name parameter specifies the name of the flag or arg that is affected.  The
// bind function is the function to set the value, and valopt is optional, and if specified,
// indicates the value to set; otherwise, the value is read from the flag.
func BindIndirect[T, V any](name any, bind func(*T, V) error, valopt ...V) Action {
	return bindThunk(func(c context.Context) *T {
		return FromContext(c).Value(name).(*T)
	}, bind, valopt...)
}

func bindThunk[T, V any](thunk func(context.Context) *T, bind func(*T, V) error, valopt ...V) Action {
	switch len(valopt) {
	case 0:
		proto := &Prototype{Value: support.BindSupportedValue(new(V))}
		return Pipeline(proto, bindTiming(func(c context.Context) error {
			return bind(thunk(c), c.Value("").(V))
		}))

	case 1:
		proto := &Prototype{Value: new(bool)}
		val := valopt[0]
		return Pipeline(proto, bindTiming(func(c context.Context) error {
			return bind(thunk(c), val)
		}))
	default:
		panic("expected 0 or 1 args for valopt")
	}
}

func bindTiming(a actionFunc) Action {
	return At(ActionTiming, a)
}

// Accessory provides an action which sets up an accessory flag for the current flag or argument.
// A common pattern is that a flag has a related sibling flag that can be used to refine the value.
// For example, you might define a --recursive flag next to a FileSet argument.  When a Value
// supports an accessory flag prototype, you can use this action to activate it from its Uses pipeline.
//
// The value of name determines the accessory flag name.  There are two special cases.  If it is blank,
// then the name from the prototype will be used.  If it is "-", then it will be derived from the other flag.
// For example, in the case of the FileSet recursive flag as described earlier, if the FileSet flag were
// named "files", then the accessory flag would be named --files-recursive.
func Accessory[T any, A Action](name string, fn func(T) A, actionopt ...Action) Action {
	return ActionFunc(func(c *Context) error {
		val := c.Value("").(T)
		var proto Action
		if fn != nil {
			proto = fn(val)
		}

		f := &Flag{
			Uses: accessory(c, proto, name).Append(actionopt...),
		}
		return c.AddFlag(f)
	})
}

// Accessory0 provides an action which sets up an accessory flag for the current flag or argument.
// A common pattern is that a flag has a related sibling flag that can be used to refine the value.
//
// The value of name determines the accessory flag name.  There are two special cases.  If it is blank,
// then the name from the prototype will be used.  If it is "-", then it will be derived from the other flag.
// For example, in the case of the FileSet recursive flag as described earlier, if the FileSet flag were
// named "files", then the accessory flag would be named --files-recursive.
func Accessory0(name string, actionopt ...Action) Action {
	return actionFunc(func(c context.Context) error {
		f := &Flag{
			Uses: accessory(c, nil, name).Append(actionopt...),
		}
		return Do(c, AddFlag(f))
	})
}

func accessory(ctx context.Context, proto Action, name string) ActionPipeline {
	c := FromContext(ctx)
	var (
		dep = withoutDecorators(c.Name())

		ensureCategory = Prototype{
			Category: c.option().category(),
		}
		ensureName = func() Action {
			switch name {
			case "":
				return nil
			case "-":
				return ActionFunc(func(c *Context) error {
					return c.SetName(dep + "-" + withoutDecorators(c.Name()))
				})
			default:
				return Named(name)
			}
		}()
	)
	return Pipeline(Data("DependentFlag", dep), proto, ensureCategory, ensureName)
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

// Mutex validates that explicit values are used mutually exclusively.
// When used on any flag in a mutex group, the other named flags are not allowed to be
// used.
func Mutex(names ...string) Action {
	return At(ValidatorTiming, ActionFunc(func(c *Context) error {
		if c.Seen("") {
			alsoSeen := make([]string, 0, len(names))
			for _, o := range names {
				if c.Seen(o) {
					alsoSeen = append(alsoSeen, optionName(o))
				}
			}

			switch len(alsoSeen) {
			case 0:
				return nil
			case 1:
				return fmt.Errorf("either %s or %s can be used, but not both", c.Name(), alsoSeen[0])
			case 2:
				return fmt.Errorf("can't use %s together with %s or %s", c.Name(), alsoSeen[0], alsoSeen[1])
			default:
				y := len(alsoSeen) - 1
				return fmt.Errorf("can't use %s together with %s", c.Name(), strings.Join(alsoSeen[0:y], ", ")+", or "+alsoSeen[y])
			}
		}

		return nil
	}))
}

// Data sets metadata for a command, flag, arg, or expression.  This handler is generally
// set up inside a Uses pipeline.
// When value is nil, the corresponding
// metadata is deleted
func Data(name string, value any) Action {
	return actionThunk2((*Context).SetData, name, value)
}

// Category sets the category of a command, flag, or expression.  This handler is generally
// set up inside a Uses pipeline.
func Category(name string) Action {
	return actionThunk1((*Context).SetCategory, name)
}

// Named sets the name of a command, flag, or expression.  This handler is generally
// set up inside a Uses pipeline.
func Named(name string) Action {
	return actionThunk1((*Context).SetName, name)
}

// Alias sets the given alias on the flag or command.  For args, the action is ignored
func Alias(a ...string) Action {
	return actionThunkVar1((*Context).AddAlias, a)
}

// Description sets the description of a command, flag, or expression.  This handler is generally
// set up inside a Uses pipeline.  The type should be string or fmt.Stringer
// Consider implementing a custom Stringer value that defers calculation of the description if the
// description is expensive to calculate.  One convenient implementation is from Template.Bind or
// Template.BindFunc. The implementation can also provide:
//
//   - SortUsage()        called when the option [SortedExprs] was used, which generally
//     indicates that any expressions or other sub-details should be sorted.
func Description(v any) Action {
	return actionThunk1((*Context).SetDescription, v)
}

// HelpText sets the help text for the current target
func HelpText(s string) Action {
	return actionThunk1((*Context).SetHelpText, s)
}

// UsageText sets the help text for the current target
func UsageText(s string) Action {
	return actionThunk1((*Context).SetUsageText, s)
}

// ManualText sets the manual text of a command, flag, arg, or expression.
func ManualText(s string) Action {
	return actionThunk1((*Context).SetManualText, s)
}

// Hook registers a hook that runs for any context in the given timing.
func Hook(timing Timing, handler Action) Action {
	return actionThunk2((*Context).Hook, timing, handler)
}

// HookBefore registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookBefore(pattern string, handler Action) Action {
	return actionThunk2((*Context).HookBefore, pattern, handler)
}

// HookAfter registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookAfter(pattern string, handler Action) Action {
	return actionThunk2((*Context).HookAfter, pattern, handler)
}

// HandleSignal provides an action that provides simple handling of a signal, usually os.Interrupt.
// HandleSignal updates the Context to handle the signal by exposing the context Done() channel.
// Compare this behavior to os/signal.NotifyContext.  Here's an example:
//
//	&cli.Command{
//	   Name: "command",
//	   Uses: cli.HandleSignal(os.Interrupt),
//	   Action: func(c *cli.Context) error {
//	       for {
//	           select {
//	           case <-c.Done():
//	               // Ctrl+C was called
//	               return nil
//	           default:
//	               // process another step, use return to exit
//	           }
//	       }
//	   }
//	}
//
// The signal handler is unregistered in the After pipeline.  The recommended approach
// is therefore to place cleanup into After and consider using a timeout.
// The process will be terminated when the user presses ^C for the second time:
func HandleSignal(s ...os.Signal) Action {
	return Before(actionFunc(func(c context.Context) error {
		ctx, stop := signal.NotifyContext(c, s...)
		return Do(c, Pipeline(
			SetContext(ctx),
			After(ActionOf((func())(stop))),
		))
	}))
}

// OptionalValue makes the flag's value optional, and when its value is not specified, the implied value
// is set to this value v.  Say that a flag is defined as:
//
//	&Flag {
//	  Name: "secure",
//	  Value: cli.String(),
//	  Uses: cli.OptionalValue("TLS1.2"),
//	}
//
// This example implies that --secure without a value is set to the value TLS1.2 (presumably other versions
// are allowed).  This example is a fair use case of this feature: making a flag opt-in to some sort of default
// configuration and allowing an expert configuration by using a value.
// For short options, no space can be between the flag and value (e.g. you need -sString to
// specify a String to the -s option).
// In general, making the value of a non-Boolean flag optional is not recommended when
// the command also allows arguments because it can make the syntax ambiguous.
func OptionalValue(v any) Action {
	return actionThunk1((*Context).SetOptionalValue, v)
}

// ProvideValueInitializer causes an additional child context to be created
// which is used to initialize an arbitrary value.  For more information,
// refer to the implementation provided by Context.ProvideValueInitializer.
func ProvideValueInitializer(v any, name string, actionopt ...Action) Action {
	return actionThunkVar3((*Context).ProvideValueInitializer, v, name, actionopt...)
}

// AddFlag provides an action which adds a flag to the command or app
func AddFlag(f *Flag) Action {
	return actionThunk1((*Context).AddFlag, f)
}

// AddCommand provides an action which adds a sub-command to the command or app
func AddCommand(c *Command) Action {
	return actionThunk1((*Context).AddCommand, c)
}

// AddArg provides an action which adds an arg to the command or app
func AddArg(a *Arg) Action {
	return actionThunk1((*Context).AddArg, a)
}

// AddAlias adds the specified alias to the flag or command.
func AddAlias(a string) Action {
	return Alias(a)
}

// RemoveArg provides an action which removes an arg from the command or app.
// The name specifies the name, index, or arg itself
func RemoveArg(name any) Action {
	return actionThunk1((*Context).RemoveArg, name)
}

// RemoveFlag provides an action which removes a flag from the command or app.
// The name specifies the name, index, or Flag itself
func RemoveFlag(name any) Action {
	return actionThunk1((*Context).RemoveFlag, name)
}

// RemoveCommand provides an action which removes a Command from the command or app.
// The name specifies the name, index, or Command itself
func RemoveCommand(name any) Action {
	return actionThunk1((*Context).RemoveCommand, name)
}

// RemoveAlias removes the alias from the flag or command.
func RemoveAlias(a string) Action {
	return actionThunk1((*Context).RemoveAlias, a)
}

// AddFlags provides an action which adds the specified flags to the command
func AddFlags(flags ...*Flag) Action {
	return actionThunkVar1((*Context).AddFlags, flags)
}

// AddArgs provides an action which adds the specified args to the command
func AddArgs(args ...*Arg) Action {
	return actionThunkVar1((*Context).AddArgs, args)
}

// AddCommands provides an action which adds the specified commands to the command
func AddCommands(commands ...*Command) Action {
	return actionThunkVar1((*Context).AddCommands, commands)
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
// setup.  It applies to the command in scope, so it can also be used within an
// arg or flag to affect the containing command.  The setup function fn will only be
// called in the initialization timing and only if setup hasn't been blocked by PreventSetup.
func CommandSetup(fn func(*Command)) Action {
	return commandSetupCore(false, fn)
}

func commandSetupCore(direct bool, fn func(*Command)) Action {
	return optionalSetup(func(c *Context) {
		if direct {
			if cmd, ok := c.Target().(*Command); ok {
				fn(cmd)
			}
			return
		}
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

// ImplicitValue sets the implicit value, which is a value specified for
// the arg or flag when it was not specified from the command line arguments.
// Any errors setting the value are suppressed.  In particular, note that
// the error ErrImplicitValueAlreadySet could be generated from setting value
// but it is also ignored.
func ImplicitValue(fn func(*Context) (string, bool)) Action {
	return Implicitly(ActionFunc(func(c *Context) error {
		if v, ok := fn(c); ok {
			c.SetValue(v)
		}
		return nil
	}))
}

// Implicitly runs an action when there are zero occurrences (when an implicit value is
// set)
func Implicitly(a Action) Action {
	return At(ImplicitValueTiming, actionFunc(func(c context.Context) error {
		if FromContext(c).Occurrences("") == 0 {
			return Do(c, a)
		}
		return nil
	}))
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
			if c.Occurrences("") == 0 {
				return "", false
			}
			return value, true
		}))
	})
}

// IfMatch provides an action that only runs on a matching context.
// If and only if the filter f matches will the corresponding action
// be run. If f is nil, this is a no-op.
func IfMatch(f ContextFilter, a Action) Action {
	if f == nil {
		return a
	}
	return actionFunc(func(c context.Context) error {
		if f.Matches(c) {
			return Do(c, a)
		}
		return nil
	})
}

// Assert provides an action that returns an error if the context filter does not
// match the context or otherwise runs the action. If the filter value has a method
// Describe() or String() that returns a string, it is consulted in order to produce the error
// message; otherwise, a generic error is returned.   Compare to IfMatch.
func Assert(filter ContextFilter, a Action) Action {
	return actionFunc(func(ctx context.Context) error {
		if !filter.Matches(ctx) {
			var desc string
			if d, ok := filter.(interface{ Describe() string }); ok {
				desc = d.Describe()
			} else if d, ok := filter.(fmt.Stringer); ok {
				desc = d.String()
			}
			if desc != "" {
				c := FromContext(ctx)
				return c.internalError(fmt.Errorf("context must be %s", desc))
			}
			return errAssertFailed
		}
		return Do(ctx, a)
	})
}

// Customize matches a flag, arg, or command and runs additional pipeline steps.  Customize
// is usually used to apply further customization after an extension has done setup of
// the defaults.
func Customize(pattern string, a Action) Action {
	return actionThunk2((*Context).Customize, pattern, a)
}

// Transform defines how to interpret raw values passed to a flag or arg.  The action
// is added to the Uses pipeline.  The typical use of transforms is to interpret the
// value passed to an argument as instead a reference to a file which is loaded.
// The function fn can return string, []byte, or io.Reader.  If the Value implements
// method SetData(io.Reader) error, then this is called instead when setting the Value.
// If it doesn't, then bytes or readers are read in and treated as a string and the
// usual Set method is used.  See also: FileReference and AllowFileReference, which provide
// common transforms.
func Transform(fn TransformFunc) Action {
	return actionThunk1((*Context).SetTransform, fn)
}

// ValueTransform sets the transform that applies to the syntax of the value in
// NameValue, NameValues, or Map.  The first occurrence of an unescaped equal sign is treated
// as the delimiter between the name and value (as with any of the types just mentioned).
// The value portion is transformed using the logic of the transform function.
func ValueTransform(valueFn TransformFunc) Action {
	return Transform(func(raw []string) (interface{}, error) {
		name, value, hasValue := splitValuePair(raw[1])
		if hasValue {
			values := append([]string{name, value}, raw[2:]...)
			f, err := valueFn(values)
			if err != nil {
				return nil, err
			}

			txt, err := transformOutputToString(f)
			if err != nil {
				return nil, err
			}
			return name + "=" + txt, nil
		}

		return name, nil
	})
}

func transformOutputToString(v interface{}) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case io.Reader:
		bb, err := io.ReadAll(val)
		if err != nil {
			return "", err
		}
		return string(bb), nil
	case []byte:
		return string(val), nil
	}
	panic(fmt.Sprintf("unexpected transform output %T", v))
}

// TransformFileReference obtains the transform for the given file system and whether the
// prefix @ is required.
func TransformFileReference(f fs.FS, usingAtSyntax bool) TransformFunc {
	if usingAtSyntax {
		return func(raw []string) (interface{}, error) {
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
		}
	}

	return func(raw []string) (interface{}, error) {
		readers := make([]io.Reader, len(raw)-1)
		for i, s := range raw[1:] {
			f, err := f.Open(s)
			if err != nil {
				return nil, err
			}
			readers[i] = f
		}
		return io.MultiReader(readers...), nil
	}
}

// FromEnv loads the flag or arg value from the given environment variable(s).
// Alternatively, you can set the EnvVars field on Flag or Arg to achieve the same
// behavior.
//
// The special pattern "{}" can be used to represent the long name of the
// flag transformed into uppercase, replacing internal dashes with underscores, and
// introducing an additional underscore to separate it from other text
// (hence, "APP{}" is an acceptable value to get env var APP_FLAG_NAME for the
// --flag-name flag.)
func FromEnv(vars ...string) Action {
	return Implicitly(ActionFunc(func(c *Context) error {
		if c.ImplicitlySet() {
			return nil
		}

		// Name should be long name transformed into SCREAMING_SNAKE_CASE.
		name := flagScreamingSnakeCase(c.option())
		for _, v := range vars {
			envVar := expandEnvVarName(v, name)

			// Env vars have to be present and set explicitly to non-empty string.
			// This addresses the case where Boolean flags are typically interpreted
			// as true even when empty (i.e. --bool is the same as -bool=true and perhaps
			// surprisingly --bool= ).  But ENV_VAR= is not treated as true if present.
			if val := os.Getenv(envVar); val != "" {
				c.SetValue(val)
				return nil
			}
		}
		return nil
	}))
}

func flagScreamingSnakeCase(o option) string {
	name := o.name()
	if f, ok := o.(*Flag); ok {
		name = f.LongName()
	}
	name = strings.Trim(name, "-")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToUpper(name)
}

// FromFilePath loads the value from the given file path.
// If the file does not exist or fails to load, the error is
// silently ignored.
// Alternatively, you can set the Flag or Arg field FilePath.
func FromFilePath(f fs.FS, filePath string) Action {
	return Implicitly(ActionFunc(func(c *Context) error {
		if f == nil {
			f = c.actualFS()
		}
		if c.ImplicitlySet() {
			return nil
		}
		if len(filePath) > 0 {
			data, err := fs.ReadFile(f, filePath)
			if err == nil {
				c.SetValue(string(data))
				return nil
			}
		}
		return nil
	}))
}

func expandEnvVarName(content, name string) string {
	var buf bytes.Buffer
	var prev int

	for _, idx := range patFlagName.FindAllStringIndex(content, -1) {
		buf.WriteString(content[prev:idx[0]])
		if idx[0] > 0 {
			buf.WriteString("_")
		}
		buf.WriteString(name)
		if idx[1] < len(content) {
			buf.WriteString("_")
		}
		prev = idx[1]
	}
	buf.WriteString(content[prev:])
	return buf.String()
}

// Print provides an action that formats using the default formats for its operands and writes to
// standard output using the behavior of fmt.Print.
func Print(a ...any) Action {
	return unwrapPrintFunc((*Context).Print, a)
}

// Println provides an action that formats using the default formats for its operands and writes to
// standard output using the behavior of fmt.Println
func Println(a ...any) Action {
	return unwrapPrintFunc((*Context).Println, a)
}

// Printf provides an action that formats according to a format specifier and writes to standard
// output using the behavior of fmt.Printf
func Printf(format string, a ...any) Action {
	return unwrapPrintFunc2((*Context).Printf, format, a)
}

// Fprint provides an action that formats using the default formats for its operands and writes to
// a file using the behavior of fmt.Fprint.
func Fprint(w io.Writer, a ...any) Action {
	return unwrapPrintFunc2((*Context).Fprint, w, a)
}

// Fprintln provides an action that formats using the default formats for its operands and writes to
// a file using the behavior of fmt.Fprintln
func Fprintln(w io.Writer, a ...any) Action {
	return unwrapPrintFunc2((*Context).Fprintln, w, a)
}

// Fprintf provides an action that formats according to a format specifier and writes to a file
// using the behavior of fmt.Fprintf
func Fprintf(w io.Writer, format string, a ...any) Action {
	return unwrapPrintFunc3((*Context).Fprintf, w, format, a)
}

func unwrapPrintFunc(f func(*Context, ...any) (int, error), a []any) ActionFunc {
	return func(c *Context) error {
		_, err := f(c, a...)
		return err
	}
}

func unwrapPrintFunc2[T any](f func(*Context, T, ...any) (int, error), t T, a []any) ActionFunc {
	return func(c *Context) error {
		_, err := f(c, t, a...)
		return err
	}
}

func unwrapPrintFunc3[T, U any](f func(*Context, T, U, ...any) (int, error), t T, u U, a []any) ActionFunc {
	return func(c *Context) error {
		_, err := f(c, t, u, a...)
		return err
	}
}

func (t Timing) Matches(ctx context.Context) bool {
	c := FromContext(ctx)
	switch t {
	case InitialTiming:
		return c.IsInitializing()
	case AfterTiming:
		return c.IsAfter()
	case ActionTiming:
		return c.IsAction()
	case BeforeTiming:
	}
	return c.IsBefore()
}

// String produces a textual representation of the timing
func (t Timing) String() string {
	return timingLabels[t][1]
}

// MarshalText provides the textual representation
func (t Timing) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText converts the textual representation
func (t *Timing) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, y := range timingLabels {
		if token == y[1] {
			*t = k
			return nil
		}
	}
	return nil
}

// Describe produces a representation of the timing suitable for use in messages
func (t Timing) Describe() string {
	return timingLabels[t][0]
}

func (p Prototype) Execute(ctx context.Context) error {
	return Do(ctx, Pipeline(FlagSetup(p.copyToFlag), ArgSetup(p.copyToArg), commandSetupCore(true, p.copyToCommand), p.Setup))
}

func (p Prototype) Use(action Action) Prototype {
	p.Setup = p.Setup.Use(action)
	return p
}

func (p *Prototype) copyToCommand(o *Command) {
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
	if o.Description == "" || o.Description == nil {
		o.Description = p.Description
	}
	if o.Completion == nil {
		o.Completion = p.Completion
	}

	o.Options |= p.Options
	maps.Copy(o.Data, p.Data)
	o.Aliases = append(o.Aliases, p.Aliases...)
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
	if o.Description == "" || o.Description == nil {
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
	if o.NArg == nil {
		o.NArg = p.NArg
	}
	if p.Value != nil && (o.internalFlags().destinationImplicitlyCreated() || o.Value == nil) {
		o.Value = p.Value
		o.setInternalFlags(internalFlagDestinationImplicitlyCreated, false)
	}

	o.EnvVars = append(o.EnvVars, p.EnvVars...)
	o.Options |= p.Options
	maps.Copy(o.Data, p.Data)
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
	if o.Description == "" || o.Description == nil {
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
	if p.Value != nil && (o.internalFlags().destinationImplicitlyCreated() || o.Value == nil) {
		o.Value = p.Value
		o.setInternalFlags(internalFlagDestinationImplicitlyCreated, false)
	}

	o.Aliases = append(o.Aliases, p.Aliases...)
	o.EnvVars = append(o.EnvVars, p.EnvVars...)
	o.Options |= p.Options
	maps.Copy(o.Data, p.Data)
}

// Execute the action by calling the function
func (af ActionFunc) Execute(ctx context.Context) error {
	if af == nil {
		return nil
	}
	return af(FromContext(ctx))
}

func (af actionFunc) Execute(ctx context.Context) error {
	if af == nil {
		return nil
	}
	return af(ctx)
}

// Append appends actions to the pipeline
func (p ActionPipeline) Append(x ...Action) ActionPipeline {
	return append(p, x...)
}

// Prepend prepends actions to the pipeline
func (p ActionPipeline) Prepend(x ...Action) ActionPipeline {
	return append(x, p...)
}

// Execute the pipeline by calling each action successively
func (p ActionPipeline) Execute(c context.Context) (err error) {
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

func (o *cons) Execute(c context.Context) error {
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

func (p *actionPipelines) pipeline(t Timing) Action {
	switch t {
	case AfterTiming:
		return p.After
	case BeforeTiming:
		return p.Before
	case InitialTiming:
		return p.Initializers
	case ActionTiming:
		return p.Action
	default:
		panic("unreachable!")
	}
}

func (w withTimingWrapper) Execute(ctx context.Context) error {
	c := FromContext(ctx)
	if w.t == ImplicitValueTiming {
		return c.act(Pipeline(
			setInternalFlag(internalFlagImplicitTimingActive),
			w.Action,
			unsetInternalFlag(internalFlagImplicitTimingActive),
		), BeforeTiming, false)
	}

	return c.act(w.Action, w.t, false)
}

func (b beforePipeline) Execute(c context.Context) error {
	return ActionPipeline(append(append(append(b[0], b[1]...), b[2]...), b[3]...)).Execute(c)
}

func (i *hooksSupport) hook(at Timing, a Action) error {
	i.hooks.add(at, a)
	return nil
}

func (i *hooksSupport) executeBeforeHooks(target context.Context) error {
	return Do(target, i.hooks.Before)
}

func (i *hooksSupport) executeAfterHooks(target context.Context) error {
	return Do(target, i.hooks.After)
}

func (i *hooksSupport) executeInitializeHooks(target context.Context) error {
	return Do(target, i.hooks.Initializers)
}

func (i *hooksSupport) valueTargets() []*valueTarget {
	return i.values
}

func (i *hooksSupport) addValueTarget(v *valueTarget) {
	i.values = append(i.values, v)
}

func (s *pipelinesSupport) uses() *actionPipelines {
	return &s.p
}

func (s *pipelinesSupport) appendAction(t Timing, ah Action) {
	s.p.add(t, ah)
}

func (i *internalFlagsSupport) setInternalFlags(f internalFlags, v bool) {
	if v {
		i.flags |= f
	} else {
		i.flags &= ^f
	}
}

func (i *internalFlagsSupport) internalFlags() internalFlags {
	return i.flags
}

func (m middlewareFunc) Execute(c context.Context) error {
	return m.ExecuteWithNext(c, nil)
}

func (m middlewareFunc) ExecuteWithNext(ctx context.Context, a Action) error {
	c := FromContext(ctx)
	return m(c, a)
}

func (v ValidatorFunc) Execute(ctx context.Context) error {
	return Do(ctx, At(ValidatorTiming, ActionFunc(func(c *Context) error {
		occur := c.RawOccurrences("")
		if err := v(occur); err != nil {
			return argTakerError(c.Name(), "", err, nil)
		}

		return nil
	})))
}

func (t TransformFunc) Execute(ctx context.Context) error {
	return Do(ctx, Transform(t))
}

func (w legacyActionWrapper) Execute(ctx context.Context) error {
	return w.a.Execute(FromContext(ctx))
}

func setData(data map[string]interface{}, name string, v interface{}) map[string]interface{} {
	if v == nil {
		delete(data, name)
		return data
	}
	if data == nil {
		return map[string]interface{}{
			name: v,
		}
	}
	data[name] = v
	return data
}

// actions provides a pipeline without flattening
func actions(actions ...Action) ActionPipeline {
	return ActionPipeline(actions)
}

func emptyActionImpl(*Context) error {
	return nil
}

// actionThunk1 is a higher order function that takes an argument
// and creates an Action by currying on the specified argument
func actionThunk1[T any](fn func(*Context, T) error, arg T) ActionFunc {
	return func(c *Context) error {
		return fn(c, arg)
	}
}

func actionThunkVar1[T any](fn func(*Context, ...T) error, t []T) Action {
	return ActionFunc(func(c *Context) error {
		return fn(c, t...)
	})
}

func actionThunk2[T, U any](fn func(*Context, T, U) error, arg1 T, arg2 U) Action {
	return ActionFunc(func(c *Context) error {
		return fn(c, arg1, arg2)
	})
}

func actionThunkVar3[T, U, V any](fn func(*Context, T, U, ...V) error, arg1 T, arg2 U, arg3 ...V) Action {
	return ActionFunc(func(c *Context) error {
		return fn(c, arg1, arg2, arg3...)
	})
}

func execute(c context.Context, af Action) error {
	if af == nil {
		return nil
	}
	return af.Execute(c)
}

func doThenExit(a Action) Action {
	return actionFunc(func(c context.Context) error {
		err := a.Execute(c)
		if err != nil {
			return err
		}
		return Exit(0)
	})
}

func defaultVersionFlag() *Flag {
	return &Flag{
		Uses: PrintVersion(),
	}
}

func defaultHelpFlag() *Flag {
	return &Flag{
		Name:     "help",
		Aliases:  []string{"h"},
		HelpText: "Display this help screen then exit",
		Value:    Bool(),
		Options:  Exits,
		Action:   displayHelp,
	}
}

func defaultHelpCommand() *Command {
	return &Command{
		Name:     "help",
		Aliases:  []string{"h"},
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
}
func defaultVersionCommand() *Command {
	return &Command{
		Uses: PrintVersion(),
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
