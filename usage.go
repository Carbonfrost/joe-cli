package cli

import (
	"bytes"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/juju/ansiterm"
	"github.com/juju/ansiterm/tabwriter"
	"golang.org/x/term"
)

var (
	usagePattern = regexp.MustCompile(`{(.+?)}`)
	controlCodes = regexp.MustCompile("\u001B\\[\\d+m")
)

// Template provides a wrapper around text templates to support some additional configuration
type Template struct {
	*template.Template

	// Debug when set will render errors to stderr.  This value is typically activated via the
	// environment variable CLI_DEBUG_TEMPLATES=1
	Debug bool
}

//counterfeiter:generate . Writer

// Writer provides a terminal output writer which can provide access to color and styles
type Writer interface {
	io.Writer
	io.StringWriter

	// ClearStyle removes the corresponding style
	ClearStyle(Style)
	// Reset will reset to the default style
	Reset()
	// SetColorCapable changes whether or not the writer should use color and style control codes
	SetColorCapable(bool)
	// ResetColorCapable uses autodetect to apply the default
	ResetColorCapable()
	// SetBackground updates the background color
	SetBackground(Color)
	// SetForeground updates the foreground color
	SetForeground(Color)
	// SetStyle updates the style
	SetStyle(Style)
}

// Color of terminal output
type Color = ansiterm.Color

// Style of terminal output
type Style = ansiterm.Style

type stringHelper struct {
	*ansiterm.Writer
}

type buffer struct {
	Writer
	res *bytes.Buffer
}

const (
	Bold          Style = ansiterm.Bold
	Faint               = ansiterm.Faint
	Italic              = ansiterm.Italic
	Underline           = ansiterm.Underline
	Blink               = ansiterm.Blink
	Reverse             = ansiterm.Reverse
	Strikethrough       = ansiterm.Strikethrough
	Conceal             = ansiterm.Conceal
)

const (
	Default       = ansiterm.Default
	Black         = ansiterm.Black
	Red           = ansiterm.Red
	Green         = ansiterm.Green
	Yellow        = ansiterm.Yellow
	Blue          = ansiterm.Blue
	Magenta       = ansiterm.Magenta
	Cyan          = ansiterm.Cyan
	Gray          = ansiterm.Gray
	DarkGray      = ansiterm.DarkGray
	BrightRed     = ansiterm.BrightRed
	BrightGreen   = ansiterm.BrightGreen
	BrightYellow  = ansiterm.BrightYellow
	BrightBlue    = ansiterm.BrightBlue
	BrightMagenta = ansiterm.BrightMagenta
	BrightCyan    = ansiterm.BrightCyan
	White         = ansiterm.White
)

func newBuffer() *buffer {
	res := new(bytes.Buffer)
	return &buffer{
		Writer: NewWriter(res),
		res:    res,
	}
}

// NewWriter creates a new writer
func NewWriter(w io.Writer) Writer {
	return &stringHelper{ansiterm.NewWriter(w)}
}

type usage struct {
	exprs []expr
}

type expr interface {
	exprSigil()
}

type placeholderExpr struct {
	name string
	pos  int
}

type literal struct {
	text string
}

type placeholdersByPos []*placeholderExpr

// SetColor enables or disables color output on stdout.
func SetColor(enabled bool) Action {
	return ActionFunc(func(c *Context) error {
		// It is possible if this is called early, it will need to be
		// deferred
		if c.IsInitializing() && c.Stdout == nil {
			return c.Before(SetColor(true))
		}

		c.SetColor(enabled)
		return nil
	})
}

// AutodetectColor resets whether color output is used on stdout
// to use auto-detection
func AutodetectColor() Action {
	return ActionOf((*Context).AutodetectColor)
}

// DisplayHelpScreen displays the help screen for the specified command.  If the command
// is nested, each sub-command is named.
func DisplayHelpScreen(command ...string) ActionFunc {
	return func(c *Context) error {
		current := c.App().createRoot()
		persistentFlags := make([]*Flag, 0)

		// Find command and accumulate persistent flags
		for i, c := range command {
			if i < len(command) {
				persistentFlags = append(persistentFlags, current.VisibleFlags()...)
			}

			var ok bool
			current, ok = current.Command(c)
			if !ok {
				return commandMissing(c)
			}
		}

		tpl := c.Template("help")
		lineage := ""

		if len(command) > 0 {
			all := make([]string, 0)
			if len(c.App().Name) > 0 {
				all = append(all, c.App().Name)
			}
			all = append(all, command[0:len(command)-1]...)
			lineage = " " + strings.Join(all, " ")
		}

		data := struct {
			SelectedCommand *commandData
			App             *App
		}{
			SelectedCommand: commandAdapter(current).withLineage(lineage, persistentFlags),
			App:             c.App(),
		}

		w := ansiterm.NewTabWriter(c.Stderr, 1, 8, 2, ' ', tabwriter.StripEscape)

		_ = tpl.Execute(w, data)
		_ = w.Flush()
		return nil
	}
}

// PrintVersion displays the version string.  The VersionTemplate provides the Go template
func PrintVersion() ActionFunc {
	return RenderTemplate("version", nil)
}

func defaultData(c *Context) interface{} {
	return struct {
		App     *App
		Command *Command
	}{
		App:     c.App(),
		Command: c.Command(),
	}
}

// RenderTemplate provides an action that renders the specified template using the factory function that
// creates the data that is passed to the template
func RenderTemplate(name string, data func(*Context) interface{}) ActionFunc {
	return func(c *Context) error {
		return c.RenderTemplate(name, data)
	}
}

// RegisterTemplate will register the specified template by name.
func RegisterTemplate(name string, template string) Action {
	return ActionFunc(func(c *Context) error {
		c.RegisterTemplate(name, template)
		return nil
	})
}

// RegisterTemplateFunc will register the specified function for use in template rendering.
func RegisterTemplateFunc(name string, fn interface{}) Action {
	return ActionFunc(func(c *Context) error {
		c.RegisterTemplateFunc(name, fn)
		return nil
	})
}

// RenderTemplate provides an action that renders the specified template using the factory function that
// creates the data that is passed to the template
func (c *Context) RenderTemplate(name string, data func(*Context) interface{}) error {
	tpl := c.Template(name)
	if data == nil {
		data = defaultData
	}
	return tpl.Execute(c.Stdout, data(c))
}

// RegisterTemplate will register the specified template by name.
func (c *Context) RegisterTemplate(name string, template string) {
	c.App().ensureTemplates()[name] = template
}

// RegisterTemplateFunc will register the specified function for use in template rendering.
func (c *Context) RegisterTemplateFunc(name string, fn interface{}) {
	c.App().ensureTemplateFuncs()[name] = fn
}

func defaultHelpCommand() *Command {
	return &Command{
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
}

func defaultHelpFlag() *Flag {
	return &Flag{
		Name:     "help",
		HelpText: "Display this help screen then exit",
		Value:    Bool(),
		Options:  Exits,
		Action:   displayHelp,
	}
}

func defaultVersionFlag() *Flag {
	return &Flag{
		Name:     "version",
		HelpText: "Print the build version then exit",
		Value:    Bool(),
		Options:  Exits,
		Action:   PrintVersion(),
	}
}

func defaultVersionCommand() *Command {
	return &Command{
		Name:     "version",
		HelpText: "Print the build version then exit",
		Action:   PrintVersion(),
	}
}

// Execute the template
func (t *Template) Execute(wr io.Writer, data interface{}) error {
	err := t.Template.Execute(wr, data)
	if err != nil && t.Debug {
		log.Fatal(err)
	}
	return err
}

func (u *usage) Placeholders() []string {
	res := make([]string, 0)
	for _, e := range u.placeholders() {
		res = append(res, e.name)
	}
	return res
}

func (p placeholdersByPos) Less(i, j int) bool {
	return p[i].pos < p[j].pos
}

func (p placeholdersByPos) Len() int {
	return len(p)
}

func (p placeholdersByPos) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (u *usage) placeholders() []*placeholderExpr {
	res := make(placeholdersByPos, 0, len(u.exprs))
	seen := map[string]bool{}
	for _, item := range u.exprs {
		if e, ok := item.(*placeholderExpr); ok {
			if !seen[e.name] {
				res = append(res, e)
				seen[e.name] = true
			}
		}
	}
	sort.Sort(res)
	return res
}

func (u *usage) WithoutPlaceholders() string {
	var b bytes.Buffer
	for _, e := range u.exprs {
		switch item := e.(type) {
		case *placeholderExpr:
			b.WriteString(item.name)
		case *literal:
			b.WriteString(item.text)
		}
	}
	return b.String()
}

func (b *buffer) String() string {
	return b.res.String()
}

func (*placeholderExpr) exprSigil() {}
func (*literal) exprSigil()         {}

func bold(s string) string {
	res := newBuffer()
	res.SetStyle(Bold)
	res.WriteString(s)
	res.Reset()
	return res.String()
}

// bold command name, flag names
// underline arg names, value placeholders
func sprintSynopsisTokens(c *commandSynopsis, enableColor bool) []string {
	tokens := make([]string, 0)

	var (
		add = func(s string) {
			tokens = append(tokens, s)
		}
	)
	add(bold(c.name))

	groups := c.flags
	if len(groups[actionGroup]) > 0 {
		res := newBuffer()
		res.WriteString("{")
		for i, f := range groups[actionGroup] {
			if i > 0 {
				res.WriteString(" | ")
			}
			f.write(res, true)
		}
		res.WriteString("}")
		add(res.String())
	}

	// short option list -abc
	if len(groups[onlyShortNoValue]) > 0 {
		res := newBuffer()
		res.WriteString("-")
		for _, f := range groups[onlyShortNoValue] {
			res.WriteString(f.short)
		}
		add(res.String())
	}

	if len(groups[onlyShortNoValueOptional]) > 0 {
		res := newBuffer()
		res.WriteString("[-")
		for _, f := range groups[onlyShortNoValueOptional] {
			res.WriteString(f.short)
		}
		res.WriteString("]")
		add(res.String())
	}

	for _, f := range groups[otherOptional] {
		res := newBuffer()
		res.WriteString("[")
		f.write(res, true)
		res.WriteString("]")
		add(res.String())
	}

	for _, f := range groups[other] {
		res := newBuffer()
		f.write(res, true)
		add(res.String())
	}

	if c.rtl {
		var start int
		for i, p := range c.args {
			if p.optional {
				start = i
				break
			}
			add(p.String())
		}

		optionalArgs := c.args[start:]
		open := strings.Repeat("[", len(optionalArgs))
		for i, p := range optionalArgs {
			if i == 0 {
				add(open + p.String() + "]")
			} else {
				add(p.String() + "]")
			}
		}
	} else {

		for _, a := range c.args {
			if a.optional {
				add("[" + a.String() + "]")
			} else {
				add(a.String())
			}
		}
	}

	return tokens
}

func (u *usage) helpText() string {
	var b bytes.Buffer
	w := NewWriter(&b)

	for _, e := range u.exprs {
		switch item := e.(type) {
		case *placeholderExpr:
			w.SetStyle(Underline)
			w.WriteString(item.name)
			w.Reset()
		case *literal:
			b.WriteString(item.text)
		}
	}
	return b.String()
}

func (w *stringHelper) WriteString(s string) (int, error) {
	return w.Writer.Write([]byte(s))
}

func (w *stringHelper) ResetColorCapable() {
	enabled := func(w io.Writer) bool {
		f, ok := w.(*os.File)
		if !ok {
			return false
		}

		// https://no-color.org/, which requires any value to be treated as true
		if _, ok := os.LookupEnv("NO_COLOR"); ok {
			return false
		}
		return os.Getenv("TERM") != "dumb" && term.IsTerminal(int(f.Fd()))
	}

	w.SetColorCapable(enabled(w.Writer))
}

func sprintSynopsis(t valueWritesSynopsis, enableColor bool) string {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	if enableColor {
		_, ok := os.LookupEnv("NO_COLOR")
		enableColor = !ok
	}

	w.SetColorCapable(enableColor)
	t.WriteSynopsis(w)
	return buf.String()
}

func parseUsage(text string) *usage {
	content := []byte(text)
	allIndexes := usagePattern.FindAllSubmatchIndex(content, -1)
	result := []expr{}

	var index int
	for _, loc := range allIndexes {
		if index < loc[0] {
			result = append(result, newLiteral(content[index:loc[0]]))
		}
		key := content[loc[2]:loc[3]]
		result = append(result, newExpr(key))
		index = loc[1]
	}
	if index < len(content) {
		result = append(result, newLiteral(content[index:]))
	}

	return &usage{
		result,
	}
}

func newLiteral(token []byte) expr {
	return &literal{string(token)}
}

func newExpr(token []byte) expr {
	positionAndName := strings.SplitN(string(token), ":", 2)
	if len(positionAndName) == 1 {
		return &placeholderExpr{name: positionAndName[0], pos: -1}
	}

	pos, _ := strconv.Atoi(positionAndName[0])
	name := positionAndName[1]
	return &placeholderExpr{name: name, pos: pos}
}

func lenIgnoringCSI(s string) int {
	return len(controlCodes.ReplaceAllString(s, ""))
}

func displayHelp(c *Context) error {
	command := make([]string, 0)

	// Ignore any flags that were detected in this context
	for _, c := range c.List("command") {
		if c[0] == '-' {
			continue
		}
		command = append(command, c)
	}
	return c.Do(DisplayHelpScreen(command...))
}
