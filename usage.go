package cli

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/Carbonfrost/joe-cli/internal/synopsis"
	"github.com/juju/ansiterm"
	"github.com/juju/ansiterm/tabwriter"
	"golang.org/x/term"
)

var (
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
	// ResetColorCapable uses auto detect to apply the default
	ResetColorCapable()
	// ColorCapable gets whether the writer is capable of writing color and style
	ColorCapable() bool
	// SetBackground updates the background color
	SetBackground(Color)
	// SetForeground updates the foreground color
	SetForeground(Color)
	// SetStyle updates the style
	SetStyle(Style)
	// Underline writes values in underline (if available)
	Underline(...any) (int, error)
	// Bold writes values in bold (if available)
	Bold(...any) (int, error)
	// Styled writes values in corresponding style (if available)
	Styled(Style, ...any) (int, error)
}

// Color of terminal output
type Color = ansiterm.Color

// Style of terminal output
type Style = ansiterm.Style

type stringHelper struct {
	*ansiterm.Writer
	enabled bool
}

type templateBinding struct {
	t    *Template
	data func() interface{}
}

type buffer struct {
	Writer
	res *bytes.Buffer
}

type wrapper struct {
	io.Writer

	pending bytes.Buffer
	Limit   int
	Indent  string
}

type synopsisWrapper[T synopsis.Stringer] struct {
	s T
}

// ANSI terminal styles
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

// ANSI terminal colors
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

// NewBuffer creates a buffer which is a Writer that can be used to accumulate
// text into a buffer.  Color is enabled using auto-detection on stdout.
func NewBuffer() *buffer {
	res := new(bytes.Buffer)
	w := NewWriter(res)
	w.SetColorCapable(colorEnabled(os.Stdout))
	return &buffer{
		Writer: w,
		res:    res,
	}
}

// NewBuffer creates a buffer which is a Writer that can be used to accumulate
// text into a buffer.  Color is enabled depending upon whether it has been enabled
// for stdout.
func (c *Context) NewBuffer() *buffer {
	res := NewBuffer()
	res.SetColorCapable(colorEnabled(c.Stdout))
	return res
}

// NewWriter creates a new writer with support for color if TTY is detected
func NewWriter(w io.Writer) Writer {
	return &stringHelper{
		Writer:  ansiterm.NewWriter(w),
		enabled: colorEnabled(w),
	}
}

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
func DisplayHelpScreen(command ...string) Action {
	return ActionFunc(func(c *Context) error {
		current := c.Root().Command()

		var appName string
		if current.fromApp != nil {
			appName = current.fromApp.Name
		}

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

		tpl := c.Template("Help")
		if tpl == nil {
			panic("help template not registered")
		}
		lineage := ""

		if len(command) > 0 {
			all := make([]string, 0)
			if len(appName) > 0 {
				all = append(all, appName)
			}
			all = append(all, command[0:len(command)-1]...)
			lineage = strings.Join(all, " ")
		}

		data := struct {
			SelectedCommand *commandData
			App             *App
			Debug           bool
		}{
			SelectedCommand: commandAdapter(current).withLineage(lineage, persistentFlags),
			App:             c.App(),
			Debug:           tpl.Debug,
		}

		w := ansiterm.NewTabWriter(c.Stderr, 1, 8, 2, ' ', tabwriter.StripEscape)

		_ = tpl.Execute(w, data)
		_ = w.Flush()
		return nil
	})
}

// PrintVersion displays the version string.  The VersionTemplate provides the Go template
func PrintVersion() Action {
	return Pipeline(&Prototype{
		Name:     "version",
		HelpText: "Print the build version then exit",
		Value:    Bool(),
		Options:  Exits,
	}, At(ActionTiming, ExecuteTemplate("Version", nil)))
}

// PrintLicense displays the license.  The LicenseTemplate provides the Go template
func PrintLicense() Action {
	return Pipeline(&Prototype{
		Name:     "license",
		HelpText: "Display the license and exit",
		Options:  Exits,
	}, At(ActionTiming, ExecuteTemplate("License", nil)))
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

// ExecuteTemplate provides an action that renders the specified template using the factory function that
// creates the data that is passed to the template
func ExecuteTemplate(name string, data func(*Context) any) Action {
	return actionThunk2((*Context).ExecuteTemplate, name, data)
}

// RegisterTemplate will register the specified template by name.
func RegisterTemplate(name string, template string) Action {
	return actionThunk2((*Context).RegisterTemplate, name, template)
}

// RegisterTemplateFunc will register the specified function for use in template rendering.
func RegisterTemplateFunc(name string, fn interface{}) Action {
	return actionThunk2((*Context).RegisterTemplateFunc, name, fn)
}

// ExecuteTemplate provides an action that renders the specified template using the factory function that
// creates the data that is passed to the template
func (c *Context) ExecuteTemplate(name string, data func(*Context) any) error {
	tpl := c.Template(name)
	if tpl == nil {
		return fmt.Errorf("template does not exist: %q", name)
	}
	if data == nil {
		data = defaultData
	}
	return tpl.Execute(c.Stdout, data(c))
}

// RegisterTemplate will register the specified template by name.
// The nested templates defined within the template will also be
// registered, replacing any templates that were previously defined.
// If the template definition only contains nested template definitions,
// name should be left blank.
func (c *Context) RegisterTemplate(name string, tpl string) error {
	scope := c.root().ensureTemplates()
	p, err := scope.New(name).Parse(tpl)
	if err != nil {
		return err
	}

	// Copy detected templates into the global context
	for _, inner := range p.Templates() {
		scope.AddParseTree(inner.Name(), inner.Tree)
	}

	return nil
}

// RegisterTemplateFunc will register the specified function for use in template rendering.
// Templates are stored globally at the application level.
// Though part of its signature, this function never returns an error.
func (c *Context) RegisterTemplateFunc(name string, fn any) error {
	c.root().ensureTemplateFuncs()[name] = fn
	return nil
}

// Wrap wraps the given text using a maximum line width and indentation.
// Wrapping text using this method is aware of ANSI escape sequences.
func Wrap(w io.Writer, text string, indent string, width int) {
	f := &wrapper{
		Writer: w,
		Limit:  width,
		Indent: indent,
	}
	_, _ = f.Write([]byte(text))
	_ = f.Close()
}

func (w *wrapper) Write(b []byte) (int, error) {
	if w.Limit == 0 {
		return w.Writer.Write(b)
	}

	s := w.pending.String() + string(b)
	w.pending.Reset()

	var (
		ansi      bool
		userSpace = true
		buf       bytes.Buffer

		// lengths are based on printable rune widths
		lineLen int

		tryWrite = func(from *bytes.Buffer, len int) (res bool) {
			res = lineLen+len < w.Limit
			if res {
				lineLen += len
				from.WriteTo(w.Writer)
				from.Reset()
			}
			return
		}
	)

	for _, c := range s {
		switch {
		case c == '\x1B':
			// start ANSI escape sequence
			_, _ = buf.WriteRune(c)
			ansi = true

		case ansi:
			_, _ = buf.WriteRune(c)
			if isCSITerminator(c) {
				ansi = false
			}

		case unicode.IsSpace(c) && c != '\n':
			// This is the case were the user has placed space right after
			// a new line, which indicates that they have purposely done their
			// own indentation
			if lineLen == 0 && userSpace {
				w.pending.WriteRune(c)
				break
			}

			bufLen := printableWidth(buf.String())

			// Otherwise for non-user space, skip leading space on a new line
			if bufLen+lineLen == 0 {
				break
			}

			if tryWrite(&buf, bufLen) {
				w.pending.WriteRune(c)
				break
			}

			fallthrough

		case c == '\n':
			lineLen = 0
			buf.WriteTo(w.Writer)
			buf.Reset()
			w.Writer.Write([]byte("\n"))

			w.pending.Reset()
			w.pending.WriteString(w.Indent)
			userSpace = c == '\n' // will be false on fallthrough from previous case

		default:
			tryWrite(&w.pending, w.pending.Len())
			buf.WriteRune(c)
			userSpace = false
		}
	}

	buf.WriteTo(&w.pending)
	return len(b), nil
}

func (w *wrapper) Close() error {
	w.Write([]byte("\n"))
	return nil
}

func printableWidth(s string) int {
	var (
		n    int
		ansi bool
	)

	for _, c := range s {
		switch {
		case c == '\x1B':
			// start ANSI escape sequence
			ansi = true
		case ansi:
			if isCSITerminator(c) {
				ansi = false
			}
		default:
			n++
		}
	}

	return n
}

func isCSITerminator(c rune) bool {
	return (c >= 0x40 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a)
}

// Execute the template
func (t *Template) Execute(wr io.Writer, data interface{}) error {
	err := t.Template.Execute(wr, data)
	if err != nil && t.Debug {
		log.Fatal(err)
	}
	return err
}

// Bind the template to its data for later execution.  This generates a Stringer which can
// be called to get the contents later.  A common use of binding the template is an argument
// to an arg or flag description.
func (t *Template) Bind(data interface{}) fmt.Stringer {
	return &templateBinding{t, func() interface{} { return data }}
}

// BindFunc will bind the template to its data for later execution.  This generates a Stringer which can
// be called to get the contents later.  A common use of binding the template is an argument
// to an arg or flag description.
func (t *Template) BindFunc(data func() interface{}) fmt.Stringer {
	return &templateBinding{t, data}
}

func (t *templateBinding) String() string {
	var buf bytes.Buffer
	err := t.t.Execute(&buf, t.data())
	if err != nil {
		return err.Error()
	}
	return buf.String()
}

func (b *buffer) String() string {
	return b.res.String()
}

func (w *stringHelper) WriteString(s string) (int, error) {
	return w.Writer.Write([]byte(s))
}

func (w *stringHelper) ResetColorCapable() {
	w.SetColorCapable(colorEnabled(w.Writer))
}

func (w *stringHelper) ColorCapable() bool {
	return w.enabled
}

func (w *stringHelper) SetColorCapable(value bool) {
	w.enabled = value
	w.Writer.SetColorCapable(value)
}

func (s *stringHelper) Underline(v ...any) (int, error) {
	return s.Styled(Underline, v...)
}

func (s *stringHelper) Bold(v ...any) (int, error) {
	return s.Styled(Bold, v...)
}

func (s *stringHelper) Styled(style Style, v ...any) (int, error) {
	s.SetStyle(style)
	n, err := fmt.Fprint(s, v...)
	s.Reset()
	return n, err
}

func wrapSynopsis[T synopsis.Stringer](s T) *synopsisWrapper[T] {
	return &synopsisWrapper[T]{s}
}

func (s *synopsisWrapper[T]) String() string {
	buf := NewBuffer()
	s.s.WriteTo(buf)
	return buf.String()
}

func colorEnabled(w io.Writer) bool {
	if s, ok := w.(*stringHelper); ok {
		return s.enabled
	}

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

func sprintSynopsis(s synopsis.Stringer) string {
	buf := NewBuffer()
	buf.SetColorCapable(false)
	s.WriteTo(buf)
	return buf.String()
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
