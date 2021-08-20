package cli

import (
	"bytes"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

var (
	usagePattern = regexp.MustCompile(`{(.+?)}`)
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

type placeholdersByPos []*placeholderExpr

func DisplayHelpScreen(command ...string) ActionFunc {
	return func(c *Context) error {
		cmd, err := findCommand(c.App().createRoot(), command)
		if err != nil {
			return err
		}

		tpl := c.Template("help")
		lineage := ""

		if len(command) > 0 {
			lineage = c.App().Name + " " + strings.Join(command[0:len(command)-1], " ")
		}

		data := struct {
			SelectedCommand *commandData
			CommandLineage  string
			App             *App
		}{
			SelectedCommand: commandAdapter(cmd),
			CommandLineage:  lineage,
			App:             c.App(),
		}

		w := tabwriter.NewWriter(c.Stderr, 1, 8, 2, ' ', 0)

		_ = tpl.Execute(w, data)
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

		w := tabwriter.NewWriter(c.Stderr, 1, 8, 2, ' ', 0)

		_ = tpl.Execute(w, data)
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
		Action: func(c *Context) error {
			return c.Do(doThenExit(DisplayHelpScreen(c.List("command")...)))
		},
	}
}

func defaultHelpFlag() *Flag {
	return &Flag{
		Name:     "help",
		HelpText: "Display this help screen then exit",
		Value:    Bool(),
		Action:   doThenExit(DisplayHelpScreen()),
	}
}

func defaultVersionFlag() *Flag {
	return &Flag{
		Name:     "version",
		HelpText: "Print the build version then exit",
		Value:    Bool(),
		Action:   doThenExit(PrintVersion()),
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
