package cli

import (
	"strings"
	"text/template"
)

type commandData struct {
	Name               string
	Names              []string
	Description        string
	HelpText           string
	ManualText         string
	Synopsis           *commandSynopsis
	Lineage            string
	VisibleCommands    []*commandData
	VisibleFlags       flagDataList
	VisibleArgs        flagDataList
	Persistent         *persistentCommandData
	CommandsByCategory []*commandCategory
	FlagsByCategory    []*flagCategory
	Data               map[string]interface{}
	HangingIndent      int
}

type persistentCommandData struct {
	FlagsByCategory []*flagCategory
	VisibleFlags    flagDataList
}

type flagData struct {
	Name        string
	Synopsis    *flagSynopsis
	HelpText    string
	ManualText  string
	Description string
	Data        map[string]interface{}
}

type exprData struct {
	Name        string
	Synopsis    *exprSynopsis
	HelpText    string
	ManualText  string
	Description string
	Data        map[string]interface{}
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
	VisibleExprs []*exprData
	Data         map[string]interface{}
}

type exprDescriptionData struct {
	VisibleExprs    []*exprData
	ExprsByCategory []*exprCategory
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
{{ "\t" }}{{ Execute "FlagSynopsis" .Synopsis | ExtraSpaceBeforeFlag }}{{ "\t" }}{{.HelpText}}
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

{{- define "ExtendedDescription" -}}
{{- "\n" -}}
{{- range .VisibleArgs -}}
{{ if .Description }}{{ .Description }}{{ "\n" }}{{end}}
{{- end -}}
{{- range .VisibleFlags -}}
{{ if .Description }}{{ .Description }}{{ "\n" }}{{end}}
{{- end -}}
{{- end -}}

{{- define "PersistentFlags" -}}
{{ if .Persistent.VisibleFlags -}}
Global options (specify before any sub-commands): {{ "\n" }}
{{- template "Flags" .Persistent -}}
{{ end }}
{{- end -}}

{{/* Usage is the entry point, which calls flags, subcommands */}} 
{{- define "Usage" -}}
usage: {{ if .SelectedCommand.Lineage -}}
	{{- .SelectedCommand.Lineage -}}
	{{- " " -}}
{{- end -}}
{{ Execute "CommandSynopsis" .SelectedCommand.Synopsis | HangingIndent .SelectedCommand.HangingIndent }}

{{ if .SelectedCommand.Description }}
{{ .SelectedCommand.Description | Wrap 4 }}
{{ else if .SelectedCommand.HelpText }}
{{ .SelectedCommand.HelpText | Wrap 4 }}
{{- end -}}
{{- template "Flags" .SelectedCommand -}}
{{- template "Subcommands" .SelectedCommand -}}
{{- template "PersistentFlags" .SelectedCommand -}}
{{- template "ExtendedDescription" .SelectedCommand -}}
{{- end -}}

{{- template "Usage" $ -}}
`

	expressionTemplate string = `
{{- define "Expression" -}}
{{ "\t" }}{{ template "ExpressionSynopsis" .Synopsis }}{{ "\t" }}{{.HelpText}}
{{- end -}}


{{- define "Description" -}}
{{ if .VisibleExprs -}}
Expressions:
{{ range .ExprsByCategory }}
{{ if .Category }}{{.Category}}:{{ end }}
{{ if .Undocumented -}}
{{ .VisibleExprs.Names | Join ", " | Wrap 4 }}
{{- else -}}
{{ range .VisibleExprs }}{{- template "Expression" . -}}{{ "\n" }}{{end}}
{{- end -}}
{{- else -}}
{{ range .VisibleExprs }}{{- template "Expression" . -}}{{ "\n" }}{{end}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- template "Description" .Description -}}
`

	// VersionTemplate specifies the Go template for what is printed when
	//   the version flag or command is used.
	VersionTemplate string = "{{ .App.Name }}, version {{ .App.Version }}\n"

	builtinFuncs = template.FuncMap{
		"Join": func(v string, args []string) string {
			return strings.Join(args, v)
		},
		"Repeat": func(count int, s string) string {
			return strings.Repeat(s, count)
		},

		// These are used in the default synopsis (but you need color extension
		// to actually activate them)
		"Bold": func(s string) string {
			return s
		},
		"BoldFirst": func(s []string) []string {
			return s
		},
		"Underline": func(s string) string {
			return s
		},
		"Trim": strings.TrimSpace,
	}

	synopsisTemplate = template.Must(
		template.New("Synopsis").Funcs(builtinFuncs).Parse(`
{{- define "ArgSynopsis" -}}
	{{ .Value }}
	{{- if .Multi -}}
	    ...
	{{- end -}}
{{- end -}}

{{- define "FlagSynopsis" -}}
   {{- .Names | Join ", " | Bold }}{{ .Separator -}}
   {{- template "ValueSynopsis" .Value -}}
{{- end -}}

{{- define "FlagSynopsisPrimary" -}}
   {{- .Primary | Bold }}{{ .Separator -}}
   {{- template "ValueSynopsis" .Value -}}
{{- end -}}

{{- define "CommandSynopsis" -}}
{{- .Name | Bold -}}
	{{ with index .Flags 5 -}}{{/* actionGroup */ -}}
		{{ if . -}}
			{{- " {" -}}
				{{- range $i, $f := . -}}
					{{- if $i }} | {{ end -}}
					{{ template "FlagSynopsis" $f }}
				{{- end -}}
			{{ "}" -}}
		{{ end -}}
	{{ end -}}
	{{ with index .Flags 0 -}}{{/* onlyShortNoValue */ -}}
		{{ if . -}}
			{{- " -" -}}
				{{- range $i, $f := . -}}
					{{- $f.Short }}
				{{- end -}}
		{{ end -}}
	{{ end -}}	
	{{ with index .Flags 1 -}}{{/* onlyShortNoValueOptional */ -}}
		{{ if . -}}
			{{- " [-" -}}
				{{- range $i, $f := . -}}
					{{- $f.Short }}
				{{- end -}}
			{{- "]" -}}
		{{ end -}}
	{{ end -}}		
	{{ with index .Flags 3 -}}{{/* otherOptional */ -}}
		{{ if . -}}
				{{- range $i, $f := . -}}
					{{- " [" -}}
						{{ template "FlagSynopsisPrimary" $f }}
					{{- "]" -}}
				{{- end -}}
		{{ end -}}
	{{ end -}}		
	{{ with index .Flags 4 -}}{{/* other */ -}}
		{{ if . -}}
				{{- range $i, $f := . -}}
						{{ template "FlagSynopsisPrimary" $f }}
				{{- end -}}
		{{ end -}}
	{{ end -}}			

	{{- template "ArgList" . }}
{{- end -}}

{{- define "ArgList" -}}
	{{- range $a := .RequiredArgs -}}
		{{- " " -}}
		{{ template "ArgSynopsis" $a }}
	{{- end -}}	

	{{- if .OptionalArgs -}}
		{{- " " -}}
		{{- range $i, $a := .OptionalArgs -}}
			{{- if $.RTL  -}}
				{{- if (eq 0 $i) -}}
					{{- "[" | Repeat ($.OptionalArgs | len) -}}
				{{- else -}}
					{{- " " -}}
				{{- end -}}
			{{- else -}}
				{{- "[" -}}
			{{- end -}}	

			{{ template "ArgSynopsis" $a -}}
			{{- "]" -}}
		{{- end -}}		
	{{- end -}}			
{{- end -}}

{{- define "ExpressionSynopsis" -}}
{{- .Names | BoldFirst | Join ", " -}}
	{{- template "ArgList" . }}
{{- end -}}

{{- define "ValueSynopsis" -}}
   {{- .Placeholder | Underline -}}
{{- end -}}

`))
)

func (c *commandData) withLineage(lineage string, persistent []*Flag) *commandData {
	c.Lineage = lineage
	c.Persistent = &persistentCommandData{
		VisibleFlags:    visibleFlags(persistent),
		FlagsByCategory: visibleFlagCategories(GroupFlagsByCategory(persistent)),
	}
	c.HangingIndent = len("usage: ") + len(lineage) + 1 + len(c.Name)
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
	)

	return &commandData{
		Name:               val.Name,
		Names:              val.Names(),
		Description:        val.Description,
		HelpText:           val.HelpText,
		ManualText:         val.ManualText,
		Synopsis:           val.newSynopsis(),
		VisibleArgs:        visibleArgs(val.VisibleArgs()),
		VisibleFlags:       visibleFlags(val.VisibleFlags()),
		VisibleCommands:    visibleCommands(val.Subcommands),
		CommandsByCategory: visibleCategories(GroupedByCategory(val.Subcommands)),
		FlagsByCategory:    visibleFlagCategories(GroupFlagsByCategory(val.Flags)),
		Persistent: &persistentCommandData{
			VisibleFlags: []*flagData{},
		},
		Data: val.Data,
	}
}

func flagAdapter(val *Flag) *flagData {
	syn := val.newSynopsis()
	return &flagData{
		Name:        val.Name,
		HelpText:    syn.Value.usage.helpText(),
		ManualText:  val.ManualText,
		Description: val.Description,
		Synopsis:    syn,
		Data:        val.Data,
	}
}

func argAdapter(val *Arg) *flagData {
	return &flagData{
		Name:        val.Name,
		HelpText:    val.HelpText,
		ManualText:  val.ManualText,
		Description: val.Description,
		Data:        val.Data,
	}
}

func exprAdapter(val *Expr) *exprData {
	syn := val.newSynopsis()
	return &exprData{
		Name:        val.Name,
		HelpText:    syn.usage.helpText(),
		Description: val.Description,
		ManualText:  val.ManualText,
		Synopsis:    syn,
		Data:        val.Data,
	}
}

func exprDescription(e *Expression) *exprDescriptionData {
	exprs := e.Exprs
	var (
		visibleExprs = func(items []*Expr) []*exprData {
			res := make([]*exprData, 0, len(items))
			for _, a := range items {
				res = append(res, exprAdapter(a))
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
	return &exprDescriptionData{
		VisibleExprs:    visibleExprs(exprs),
		ExprsByCategory: visibleExprCategories(GroupExprsByCategory(exprs)),
	}
}
