package cli

import (
	"bytes"
	"fmt"
	"github.com/juju/ansiterm"
	"github.com/juju/ansiterm/tabwriter"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var (
	usagePattern = regexp.MustCompile(`{(.+?)}`)
	controlCodes = regexp.MustCompile("\u001B\\[\\d+m")
)

type usage struct {
	exprs []expr
}

type expr interface {
}

type placeholderExpr struct {
	name string
	pos  int
}

type literal struct {
	text string
}

type usageGenerator interface {
	command(c *commandSynopsis) []string
	arg(a *argSynopsis) string
	flag(f *flagSynopsis, hideAlternates bool) string
	expr(f *exprSynopsis) string
	value(v *valueSynopsis) string
	helpText(u *usage) string
}

type termFormatter struct{}

type placeholdersByPos []*placeholderExpr

type defaultUsage struct {
	*termFormatter
}

type csPair struct {
	Open  string
	Close string
}

var (
	textUsage   = &defaultUsage{} // nil formatter disables formatting
	modernUsage = &defaultUsage{&termFormatter{}}

	bold      = namedCSPair(1, 22)
	underline = namedCSPair(4, 24)
)

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

		executeTemplate(tpl, w, data)
		_ = w.Flush()
		return nil
	}
}

func PrintVersion() ActionFunc {
	return func(c *Context) error {
		tpl := c.Template("version")
		data := struct {
			App *App
		}{
			App: c.App(),
		}

		w := ansiterm.NewTabWriter(c.Stderr, 1, 8, 2, ' ', 0)

		executeTemplate(tpl, w, data)
		_ = w.Flush()
		return nil
	}
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
		Action:   doThenExit(PrintVersion()),
	}
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

// bold command name, flag names
// underline arg names, value placeholders

func (d *defaultUsage) command(c *commandSynopsis) []string {
	tokens := make([]string, 0)

	var (
		add = func(s string) {
			tokens = append(tokens, s)
		}
	)
	add(d.Bold(c.name))

	groups := c.flags
	if len(groups[actionGroup]) > 0 {
		var res bytes.Buffer
		res.WriteString("{")
		for i, f := range groups[actionGroup] {
			if i > 0 {
				res.WriteString(" | ")
			}
			res.WriteString(d.flag(f, true))
		}
		res.WriteString("}")
		add(res.String())
	}

	// short option list -abc
	if len(groups[onlyShortNoValue]) > 0 {
		var res bytes.Buffer
		res.WriteString("-")
		for _, f := range groups[onlyShortNoValue] {
			res.WriteString(f.short)
		}
		add(res.String())
	}

	if len(groups[onlyShortNoValueOptional]) > 0 {
		var res bytes.Buffer
		res.WriteString("[-")
		for _, f := range groups[onlyShortNoValueOptional] {
			res.WriteString(f.short)
		}
		res.WriteString("]")
		add(res.String())
	}

	for _, f := range groups[otherOptional] {
		add("[" + d.flag(f, true) + "]")
	}

	for _, f := range groups[other] {
		add(d.flag(f, true))
	}

	for _, a := range c.args {
		add(d.arg(a))
	}

	return tokens
}

func (d *defaultUsage) arg(a *argSynopsis) string {
	if a.multi {
		return a.value + "..."
	}
	return a.value
}

func (d *defaultUsage) flag(f *flagSynopsis, hideAlternates bool) string {
	sepIfNeeded := ""
	place := f.value.placeholder
	if len(place) > 0 {
		sepIfNeeded = f.sep
	}
	names := d.Bold(f.names(hideAlternates))
	return names + sepIfNeeded + d.Underline(place)
}

func (d *defaultUsage) expr(e *exprSynopsis) string {
	var b bytes.Buffer
	names := d.Bold(e.names())
	b.WriteString(names)
	for _, a := range e.args {
		b.WriteString(" ")
		b.WriteString(d.Underline(d.arg(a)))
	}
	return b.String()
}

func (d *defaultUsage) value(v *valueSynopsis) string {
	return d.Underline(v.placeholder)
}

func (d *defaultUsage) helpText(u *usage) string {
	var b bytes.Buffer
	for _, e := range u.exprs {
		switch item := e.(type) {
		case *placeholderExpr:
			b.WriteString(d.Underline(item.name))
		case *literal:
			b.WriteString(item.text)
		}
	}
	return b.String()
}

func (t *termFormatter) Bold(s string) string {
	if t == nil {
		return s
	}
	return bold.Open + s + bold.Close
}

func (t *termFormatter) Underline(s string) string {
	if t == nil {
		return s
	}
	return underline.Open + s + underline.Close
}

func executeTemplate(tpl *template.Template, w io.Writer, d interface{}) {
	err := tpl.Execute(w, d)
	if err != nil && os.Getenv("DEBUG_TEMPLATES") == "1" {
		log.Fatal(err)
	}
}

// namedCSPair provides the ANSI control scheme pair for a formatting sequence.
// tabwriter escapes are added to disregard them in formatting
func namedCSPair(open uint8, close uint8) csPair {
	return csPair{
		Open:  fmt.Sprintf("%[1]s\u001B[%[2]dm%[1]s", "", open),
		Close: fmt.Sprintf("%[1]s\u001B[%[2]dm%[1]s", "", close),
	}
}

func getUsageGenerator() usageGenerator {
	if os.Getenv("NO_COLOR") == "1" {
		return textUsage
	}
	return modernUsage
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
		result = append(result, newLiteral(content[index:len(content)]))
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
