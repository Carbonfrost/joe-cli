package cli

type commandData struct {
	Name               string
	Names              []string
	Description        string
	HelpText           string
	Synopsis           []string
	Lineage            string
	VisibleCommands    []*commandData
	VisibleFlags       flagDataList
	VisibleArgs        flagDataList
	VisibleExprs       flagDataList
	Persistent         *persistentCommandData
	CommandsByCategory []*commandCategory
	FlagsByCategory    []*flagCategory
	ExprsByCategory    []*exprCategory
	Data               map[string]interface{}
}

type persistentCommandData struct {
	FlagsByCategory []*flagCategory
	VisibleFlags    flagDataList
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

type flagDataList []*flagData
type flagCategory struct {
	Undocumented bool
	Category     string
	VisibleFlags flagDataList
	Data         map[string]interface{}
}

type exprCategory struct {
	Undocumented bool
	Category     string
	VisibleExprs flagDataList
	Data         map[string]interface{}
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

{{- define "Flag" -}}
{{ "\t" }}{{.Synopsis}}{{ "\t" }}{{.HelpText}}
{{- end -}}

{{- define "Flags" -}}
{{ range .FlagsByCategory }}
{{ if .Category }}{{.Category}}:{{ end }}
{{ if .Undocumented -}}
{{ .VisibleFlags.Names | Join ", " | Wrap 4 }}
{{- else -}}
{{ range .VisibleFlags }}{{- template "Flag" . -}}{{ "\n" }}{{end}}
{{- end -}}
{{- else -}}
{{ range .VisibleFlags }}{{- template "Flag" . -}}{{ "\n" }}{{end}}
{{- end }}
{{- end -}}

{{- define "PersistentFlags" -}}
{{ if .Persistent.VisibleFlags -}}
Global options (specify before any sub-commands): {{ "\n" }}
{{- template "Flags" .Persistent -}}
{{ end }}
{{- end -}}

{{- define "Expression" -}}
{{ "\t" }}{{.Synopsis}}{{ "\t" }}{{.HelpText}}
{{- end -}}

{{- define "Expressions" -}}
{{ if .VisibleExprs -}}
{{ range .ExprsByCategory }}
{{ if .Category }}{{.Category}}:{{ end }}
{{ if .Undocumented -}}
{{ .VisibleExprs.Names | Join ", " | Wrap 4 }}
{{- else -}}
{{ range .VisibleExprs }}{{- template "Expression" . -}}{{ "\n" }}{{end}}
{{- end -}}
{{- else -}}
Expressions:
{{ range .VisibleExprs }}{{- template "Expression" . -}}{{ "\n" }}{{end}}
{{- end -}}
{{- end -}}
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
{{- template "PersistentFlags" .SelectedCommand -}}
{{- end -}}

{{- template "Usage" $ -}}
`

	// VersionTemplate specifies the Go template for what is printed when
	//   the version flag or command is used.
	VersionTemplate string = "{{ .App.Name }}, version {{ .App.Version }}\n"
)

func (c *commandData) withLineage(lineage string, persistent []*Flag) *commandData {
	gen := getUsageGenerator()
	c.Lineage = lineage
	c.Persistent = &persistentCommandData{
		VisibleFlags:    visibleFlags(persistent, gen),
		FlagsByCategory: visibleFlagCategories(GroupFlagsByCategory(persistent), gen),
	}
	return c
}

func (e flagDataList) Names() []string {
	res := make([]string, 0, len(e))
	for _, x := range e {
		res = append(res, "-"+x.Name)
	}
	return res
}

func visibleFlags(items []*Flag, gen usageGenerator) []*flagData {
	res := make([]*flagData, 0, len(items))
	for _, a := range items {
		res = append(res, flagAdapter(a, gen))
	}
	return res
}

func visibleFlagCategories(items FlagsByCategory, gen usageGenerator) []*flagCategory {
	res := make([]*flagCategory, 0, len(items))
	for _, a := range items {
		res = append(res, &flagCategory{
			Category:     a.Category,
			Undocumented: a.Undocumented(),
			VisibleFlags: visibleFlags(a.VisibleFlags(), gen),
		})
	}
	if len(res) == 1 && res[0].Category == "" {
		return nil
	}
	return res
}

func commandAdapter(val *Command) *commandData {
	var (
		gen usageGenerator = getUsageGenerator()

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
				res = append(res, commandAdapter(a))
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

		visibleExprCategories = func(items ExprsByCategory) []*exprCategory {
			res := make([]*exprCategory, 0, len(items))
			for _, a := range items {
				res = append(res, &exprCategory{
					Category:     a.Category,
					Undocumented: a.Undocumented(),
					VisibleExprs: visibleExprs(a.VisibleExprs()),
				})
			}
			if len(res) == 1 && res[0].Category == "" {
				return nil
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
		VisibleFlags:       visibleFlags(val.VisibleFlags(), gen),
		VisibleExprs:       visibleExprs(val.VisibleExprs()),
		VisibleCommands:    visibleCommands(val.Subcommands),
		CommandsByCategory: visibleCategories(GroupedByCategory(val.Subcommands)),
		FlagsByCategory:    visibleFlagCategories(GroupFlagsByCategory(val.Flags), gen),
		ExprsByCategory:    visibleExprCategories(GroupExprsByCategory(val.Exprs)),
		Persistent: &persistentCommandData{
			VisibleFlags: []*flagData{},
		},
		Data: val.Data,
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
