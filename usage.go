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
	"unicode"

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

type wrapper struct {
	io.Writer

	pending bytes.Buffer
	Limit   int
	Indent  string
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

		tpl := c.Template("Help")
		lineage := ""

		if len(command) > 0 {
			all := make([]string, 0)
			if len(c.App().Name) > 0 {
				all = append(all, c.App().Name)
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
	}
}

// PrintVersion displays the version string.  The VersionTemplate provides the Go template
func PrintVersion() Action {
	return RenderTemplate("Version", nil)
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
func RenderTemplate(name string, data func(*Context) interface{}) Action {
	return ActionFunc(func(c *Context) error {
		return c.RenderTemplate(name, data)
	})
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
		if c == '\x1B' {
			// start ANSI escape sequence
			ansi = true
		} else if ansi {
			if isCSITerminator(c) {
				ansi = false
			}
		} else {
			n += 1
		}
	}

	return n
}

func isCSITerminator(c rune) bool {
	return (c >= 0x40 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a)
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
