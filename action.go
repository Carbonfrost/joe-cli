package cli

import (
	"context"
	"errors"
	"fmt"
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

// ActionPipeline represents an action composed of several steps.  To create
// this value, use the Pipeline function
type ActionPipeline struct {
	items []Action
}

type target interface {
	hookAfter(pattern string, handler Action) error
	hookBefore(pattern string, handler Action) error
	appendAction(Timing, Action)
	setCategory(name string)
	SetData(name string, v interface{})
	setInternalFlags(internalFlags)
	internalFlags() internalFlags
}

type hooksSupport struct {
	before []*hook
	after  []*hook
}

type pipelinesSupport struct {
	p *actionPipelines
}

type actionPipelines struct {
	Initializers Action // Must be strictly initializers (no automatic regrouping)
	Before       Action
	Action       Action
	After        Action
}

type actionWithTiming interface {
	Action
	timing() Timing
}

type withTimingWrapper struct {
	Action
	t Timing
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
)

var (
	emptyAction Action = ActionFunc(emptyActionImpl)

	defaultApp = actionPipelines{
		Initializers: Pipeline(
			ActionFunc(setupDefaultIO),
			ActionFunc(setupDefaultData),
			ActionFunc(setupDefaultTemplateFuncs),
			ActionFunc(setupDefaultTemplates),
			ActionFunc(addAppCommand("help", defaultHelpFlag(), defaultHelpCommand())),
			ActionFunc(addAppCommand("version", defaultVersionFlag(), defaultVersionCommand())),
		),
	}

	defaultCommand = actionPipelines{
		Before: Pipeline(
			ActionFunc(triggerBeforeFlags),
			ActionFunc(triggerBeforeArgs),
		),
		After: Pipeline(
			ActionFunc(triggerAfterArgs),
			ActionFunc(triggerAfterFlags),
		),
	}

	defaultOption = actionPipelines{
		Initializers: Pipeline(
			ActionFunc(setupValueInitializer),
		),
		Before: Pipeline(
			ActionFunc(setupOptionFromEnv),
		),
	}

	defaultExpr = actionPipelines{
		Before: Pipeline(
			ActionFunc(triggerBeforeArgs),
		),
		After: Pipeline(
			ActionFunc(triggerAfterArgs),
		),
	}

	cantHookError = errors.New("hooks are not supported in this context")
)

// Pipeline combines various actions into a single action
func Pipeline(actions ...interface{}) *ActionPipeline {
	myActions := make([]Action, len(actions))
	for i, a := range actions {
		myActions[i] = ActionOf(a)
	}

	return &ActionPipeline{myActions}
}

// Before revises the timing of the action so that it runs in the Before pipeline.
// This function is used to wrap actions in the initialization pipeline that will be
// deferred until later.
func Before(a Action) Action {
	return withTiming(a, BeforeTiming)
}

// After revises the timing of the action so that it runs in the After pipeline.
// This function is used to wrap actions in the initialization pipeline that will be
// deferred until later.
func After(a Action) Action {
	return withTiming(a, AfterTiming)
}

// Initializer marks an action handler as being for the initialization phase.  When such a handler
// is added to the Uses pipeline, it will automatically be associated correctly with the initialization
// of the value.  Otherwise, this handler is not special
func Initializer(a Action) Action {
	return withTiming(a, InitialTiming)
}

func timingOf(a Action, defaultTiming Timing) Timing {
	switch val := a.(type) {
	case actionWithTiming:
		return val.timing()
	}
	return defaultTiming
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
// Any other type causes a panic
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
	}
	panic(fmt.Sprintf("unexpected type: %T", item))
}

// ContextValue provides an action which updates the context with a
// value.
func ContextValue(key, value interface{}) Action {
	return ActionFunc(func(c *Context) error {
		c.Context = context.WithValue(c.Context, key, value)
		return nil
	})
}

// SetValue provides an action which sets the value of the flag or argument.
func SetValue(v interface{}) Action {
	return ActionFunc(func(c *Context) error {
		c.target().(option).Set(genericString(dereference(v)))
		return nil
	})
}

// Data sets metadata for a command, flag, arg, or expression.  This handler is generally
// set up inside a Uses pipeline.
func Data(name string, value interface{}) Action {
	return ActionFunc(func(c *Context) error {
		c.target().SetData(name, value)
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
		return c.target().hookBefore(pattern, handler)
	})
}

// HookAfter registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookAfter(pattern string, handler Action) Action {
	return ActionFunc(func(c *Context) error {
		return c.target().hookAfter(pattern, handler)
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
// In general, making the value of a non-Boolean flag optional is not recommended when
// the command also allows arguments because it can make the syntax ambiguous.
func OptionalValue(v interface{}) Action {
	return Initializer(ActionFunc(func(c *Context) error {
		c.Flag().setOptionalValue(v)
		return nil
	}))
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

// Execute the action by calling the function
func (af ActionFunc) Execute(c *Context) error {
	if af == nil {
		return nil
	}
	return af(c)
}

// Append appends an action to the pipeline
func (p *ActionPipeline) Append(x Action) *ActionPipeline {
	return &ActionPipeline{
		items: append(p.items, unwind(x)...),
	}
}

// Execute the pipeline by calling each action successively
func (p *ActionPipeline) Execute(c *Context) (err error) {
	if p == nil {
		return nil
	}
	for _, a := range p.items {
		err = a.Execute(c)
		if err != nil {
			return
		}
	}
	return nil
}

func (p *ActionPipeline) groupByTiming(c *Context) *actionPipelines {
	res := &actionPipelines{}
	for _, h := range p.items {
		res.add(timingOf(h, InitialTiming), h)
	}

	return res
}

func (p *actionPipelines) add(t Timing, h Action) {
	switch t {
	case InitialTiming:
		p.Initializers = pipeline(p.Initializers, h)
	case BeforeTiming:
		p.Before = pipeline(p.Before, h)
	case ActionTiming:
		p.Action = pipeline(p.Action, h)
	case AfterTiming:
		p.After = pipeline(p.After, h)
	default:
		panic("unreachable!")
	}
}

func (p *actionPipelines) exceptInitializers() *actionPipelines {
	return &actionPipelines{
		Before: p.Before,
		Action: p.Action,
		After:  p.After,
	}
}

func (p *actionPipelines) actualBefore() Action {
	if p == nil {
		return nil
	}
	return p.Before
}

func (p *actionPipelines) actualAction() Action {
	if p == nil {
		return nil
	}
	return p.Action
}

func (p *actionPipelines) actualAfter() Action {
	if p == nil {
		return nil
	}
	return p.After
}

func (w withTimingWrapper) timing() Timing {
	return w.t
}

func (i *hooksSupport) hookBefore(pat string, a Action) error {
	i.before = append(i.before, &hook{newContextPathPattern(pat), a})
	return nil
}

func (i *hooksSupport) executeBeforeHooks(target *Context) error {
	for _, b := range i.before {
		if b.pat.Match(target.Path()) {
			b.action.Execute(target)
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
			b.action.Execute(target)
		}
	}
	return nil
}

func (i *hooksSupport) append(other *hooksSupport) hooksSupport {
	return hooksSupport{
		before: append(i.before, other.before...),
		after:  append(i.after, other.after...),
	}
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

func emptyActionImpl(*Context) error {
	return nil
}

func execute(af Action, c *Context) error {
	if af == nil {
		return nil
	}
	return af.Execute(c)
}

func executeAll(c *Context, x ...Action) error {
	for _, y := range x {
		if err := execute(y, c); err != nil {
			return err
		}
	}
	return nil
}

func doThenExit(a Action) ActionFunc {
	return func(c *Context) error {
		err := a.Execute(c)
		if err != nil {
			return err
		}
		return Exit(0)
	}
}

func pipeline(x, y Action) *ActionPipeline {
	return &ActionPipeline{
		items: append(unwind(x), unwind(y)...),
	}
}

func newPipelines(uses Action, opts Option, c *Context) *actionPipelines {
	return pipeline(uses, opts.wrap()).groupByTiming(c)
}

func unwind(x Action) []Action {
	if x == nil {
		return nil
	}
	switch pipe := x.(type) {
	case *ActionPipeline:
		if pipe == nil {
			return nil
		}
		res := make([]Action, 0, len(pipe.items))
		for _, p := range pipe.items {
			res = append(res, unwind(p)...)
		}
		return res
	}
	return []Action{x}
}

var (
	_ actionWithTiming = withTimingWrapper{}
)
