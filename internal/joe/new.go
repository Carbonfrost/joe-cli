package joe

import (
	_ "embed"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/Carbonfrost/joe-cli"
	etemplate "github.com/Carbonfrost/joe-cli/extensions/template"
)

type generatorData struct {
	Name       string
	HelpText   string
	Comment    string
	Version    string
	Extensions struct {
		Color bool
		Table bool
	}
	Dependencies struct {
		Http bool
	}
}

var (
	//go:embed tpl/app.go.tpl
	appGo string

	funcs = template.FuncMap{
		"Quote": strconv.Quote,
	}

	appGoTemplate = template.Must(template.New("App").Funcs(funcs).Parse(appGo))
)

func newAppCommand() *cli.Command {
	wd, _ := os.Getwd()
	g := &generatorData{
		Name: filepath.Base(wd),
	}
	return &cli.Command{
		Name:     "app",
		HelpText: "Create a new app",
		Flags: []*cli.Flag{
			{
				Name:     "color",
				Value:    &g.Extensions.Color,
				HelpText: "Activate the color extension",
			},
			{
				Name:     "http",
				Value:    &g.Dependencies.Http,
				HelpText: "Add a dependency on joe-cli-http",
			},
			{
				Name:     "table",
				Value:    &g.Extensions.Table,
				HelpText: "Activate the table extension",
			},
			{
				Name:     "app-version",
				Aliases:  []string{"V"},
				Value:    &g.Version,
				HelpText: "Set the version string",
			},
			{
				Name:     "comment",
				Aliases:  []string{"c"},
				Value:    &g.Comment,
				HelpText: "Set the comment string",
			},
			{
				Name:     "help-text",
				Value:    &g.HelpText,
				HelpText: "Set the help text string",
			},
			{
				Name:     "name",
				Aliases:  []string{"n"},
				HelpText: "Name of the new app",
				Value:    &g.Name,
			},
		},
		Action: glueTemplateOptions(func() *etemplate.Root {
			return newAppTemplate(g)
		}),
	}
}
