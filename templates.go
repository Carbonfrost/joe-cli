package cli

type commandData struct {
	Name               string
	Names              []string
	Description        string
	HelpText           string
	ManualText         string
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
	Name       string
	Synopsis   string
	HelpText   string
	ManualText string
	Data       map[string]interface{}
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
	// HelpTemplate provides the default help Go template that is rendered on the help
	// screen
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
{{ "\t" }}{{.Synopsis | ExtraSpaceBeforeFlag }}{{ "\t" }}{{.HelpText}}
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
{{ else if .SelectedCommand.HelpText }}
{{ .SelectedCommand.HelpText | Wrap 4 }}
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
	c.Lineage = lineage
	c.Persistent = &persistentCommandData{
		VisibleFlags:    visibleFlags(persistent),
		FlagsByCategory: visibleFlagCategories(GroupFlagsByCategory(persistent)),
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

func visibleFlags(items []*Flag) []*flagData {
	res := make([]*flagData, 0, len(items))
	for _, a := range items {
		res = append(res, flagAdapter(a))
	}
	return res
}

func visibleFlagCategories(items FlagsByCategory) []*flagCategory {
	res := make([]*flagCategory, 0, len(items))
	for _, a := range items {
		res = append(res, &flagCategory{
			Category:     a.Category,
			Undocumented: a.Undocumented(),
			VisibleFlags: visibleFlags(a.VisibleFlags()),
		})
	}
	if len(res) == 1 && res[0].Category == "" {
		return nil
	}
	return res
}

func commandAdapter(val *Command) *commandData {
	var (
		visibleArgs = func(items []*Arg) []*flagData {
			res := make([]*flagData, 0, len(items))
			for _, a := range items {
				res = append(res, argAdapter(a))
			}
			return res
		}

		visibleExprs = func(items []*Expr) []*flagData {
			res := make([]*flagData, 0, len(items))
			for _, a := range items {
				res = append(res, exprAdapter(a))
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
		ManualText:         val.ManualText,
		Synopsis:           sprintSynopsisTokens(val.newSynopsis(), true),
		VisibleArgs:        visibleArgs(val.VisibleArgs()),
		VisibleFlags:       visibleFlags(val.VisibleFlags()),
		VisibleExprs:       visibleExprs(val.VisibleExprs()),
		VisibleCommands:    visibleCommands(val.Subcommands),
		CommandsByCategory: visibleCategories(GroupedByCategory(val.Subcommands)),
		FlagsByCategory:    visibleFlagCategories(GroupFlagsByCategory(val.Flags)),
		ExprsByCategory:    visibleExprCategories(GroupExprsByCategory(val.Exprs)),
		Persistent: &persistentCommandData{
			VisibleFlags: []*flagData{},
		},
		Data: val.Data,
	}
}

func flagAdapter(val *Flag) *flagData {
	syn := val.newSynopsis()
	return &flagData{
		Name:       val.Name,
		HelpText:   syn.value.usage.helpText(),
		ManualText: val.ManualText,
		Synopsis:   sprintSynopsis(val, true),
		Data:       val.Data,
	}
}

func argAdapter(val *Arg) *flagData {
	return &flagData{
		Name:       val.Name,
		HelpText:   val.HelpText,
		ManualText: val.ManualText,
		Synopsis:   sprintSynopsis(val, true),
		Data:       val.Data,
	}
}

func exprAdapter(val *Expr) *flagData {
	syn := val.newSynopsis()
	return &flagData{
		Name:       val.Name,
		HelpText:   syn.usage.helpText(),
		ManualText: val.ManualText,
		Synopsis:   sprintSynopsis(val, true),
		Data:       val.Data,
	}
}
