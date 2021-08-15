package cli

import (
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
		data := struct {
			SelectedCommand interface{}
			App             *App
		}{
			SelectedCommand: commandAdapter(cmd),
			App:             c.App(),
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

func (f *usage) Placeholders() []string {
	res := make([]string, 0)
	for _, e := range f.placeholders() {
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

func (f *usage) placeholders() []*placeholderExpr {
	res := make(placeholdersByPos, 0, len(f.exprs))
	seen := map[string]bool{}
	for _, item := range f.exprs {
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
