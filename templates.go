package cli

type commandData struct {
	Name               string
	Names              []string
	Description        string
	HelpText           string
	Synopsis           []string
	Lineage            string
	VisibleCommands    []*commandData
	VisibleFlags       []*flagData
	VisibleArgs        []*flagData
	VisibleExprs       []*flagData
	CommandsByCategory []*commandCategory
	Data               map[string]interface{}
}

type flagData struct {
	Name     string
	Synopsis string
	HelpText string
	Data     map[string]interface{}
}

type commandCategory struct {
	Category        string
	VisibleCommands []*commandData
	Data            map[string]interface{}
}

var (
	HelpTemplate = `
{{- define "Subcommands" -}}
{{ range .CommandsByCategory }}
{{ if .Category }}{{.Category}}:{{ end }}
{{- range .VisibleCommands }}
{{ "\t" }}{{ .Names | BoldFirst | Join ", " }}{{ "\t" }}{{.HelpText}}{{end}}
{{ else }}
{{- range .VisibleCommands }}
{{ "\t" }}{{.Name}}{{ "\t" }}{{.HelpText}}{{end}}
{{ end }}
{{- end -}}

{{- define "Flags" -}}
{{ range .VisibleFlags }}
{{ "\t" }}{{.Synopsis}}{{ "\t" }}{{.HelpText}}{{end}}
{{- end -}}

{{- define "Expressions" -}}
{{ range .VisibleExprs }}
{{ "\t" }}{{.Synopsis}}{{ "\t" }}{{.HelpText}}{{end}}
{{- end -}}

{{/* Usage is the entry point, which calls flags, subcommands */}} 
{{- define "Usage" -}}
usage:{{ .SelectedCommand | SynopsisHangingIndent }}
{{ if .SelectedCommand.Description }}
{{ .SelectedCommand.Description | Wrap 4 }}
{{- end -}}
{{- template "Flags" .SelectedCommand -}}
{{- template "Expressions" .SelectedCommand -}}
{{- template "Subcommands" .SelectedCommand -}}
{{- end -}}

{{- template "Usage" $ -}}
`

	// VersionTemplate specifies the Go template for what is printed when
	//   the version flag or command is used.
	VersionTemplate string = "{{ .App.Name }}, version {{ .App.Version }}\n"
)

func (c *commandData) withLineage(lineage string) *commandData {
	c.Lineage = lineage
	return c
}

func commandAdapter(val *Command, gen usageGenerator) *commandData {
	var (
		visibleFlags = func(items []*Flag) []*flagData {
			res := make([]*flagData, 0, len(items))
			for _, a := range items {
				res = append(res, flagAdapter(a, gen))
			}
			return res
		}

		visibleArgs = func(items []*Arg) []*flagData {
			res := make([]*flagData, 0, len(items))
			for _, a := range items {
				res = append(res, argAdapter(a, gen))
			}
			return res
		}

		visibleExprs = func(items []*Expr) []*flagData {
			res := make([]*flagData, 0, len(items))
			for _, a := range items {
				res = append(res, exprAdapter(a, gen))
			}
			return res
		}

		visibleCommands = func(items []*Command) []*commandData {
			res := make([]*commandData, 0, len(items))
			for _, a := range items {
				res = append(res, commandAdapter(a, gen))
			}
			return res
		}

		visibleCategories = func(items CommandsByCategory) []*commandCategory {
			res := make([]*commandCategory, 0, len(items))
			for _, a := range items {
				res = append(res, &commandCategory{
					Category:        a.Category,
					VisibleCommands: visibleCommands(a.Commands),
				})
			}
			return res
		}
	)

	return &commandData{
		Name:               val.Name,
		Names:              val.Names(),
		Description:        val.Description,
		HelpText:           val.HelpText,
		Synopsis:           gen.command(val.newSynopsis()),
		VisibleArgs:        visibleArgs(val.VisibleArgs()),
		VisibleFlags:       visibleFlags(val.VisibleFlags()),
		VisibleExprs:       visibleExprs(val.VisibleExprs()),
		VisibleCommands:    visibleCommands(val.Subcommands),
		CommandsByCategory: visibleCategories(GroupedByCategory(val.Subcommands)),
		Data:               val.Data,
	}
}

func flagAdapter(val *Flag, gen usageGenerator) *flagData {
	syn := val.newSynopsis()
	return &flagData{
		Name:     val.Name,
		HelpText: gen.helpText(syn.value.usage),
		Synopsis: gen.flag(syn, false),
		Data:     val.Data,
	}
}

func argAdapter(val *Arg, gen usageGenerator) *flagData {
	return &flagData{
		Name:     val.Name,
		HelpText: val.HelpText,
		Synopsis: gen.arg(val.newSynopsis()),
		Data:     val.Data,
	}
}

func exprAdapter(val *Expr, gen usageGenerator) *flagData {
	syn := val.newSynopsis()
	return &flagData{
		Name:     val.Name,
		HelpText: gen.helpText(syn.usage),
		Synopsis: gen.expr(syn),
		Data:     val.Data,
	}
}

func templateString(name string) string {
	switch name {
	case "version":
		return VersionTemplate
	case "help":
		return HelpTemplate
	}
	return ""
}
