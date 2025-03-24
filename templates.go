package cli

import (
	"strings"
	"text/template"

	"github.com/Carbonfrost/joe-cli/internal/synopsis"
)

type commandData struct {
	Name               string
	Names              []string
	Description        any
	HelpText           string
	ManualText         string
	Synopsis           *synopsisWrapper[*synopsis.Command]
	Lineage            string
	VisibleCommands    []*commandData
	VisibleFlags       flagDataList
	VisibleArgs        flagDataList
	Persistent         *persistentCommandData
	CommandsByCategory []*commandDataCategory
	FlagsByCategory    []*flagDataCategory
	Data               map[string]any
	HangingIndent      int
}

type persistentCommandData struct {
	FlagsByCategory []*flagDataCategory
	VisibleFlags    flagDataList
}

type flagData struct {
	Name        string
	Synopsis    *synopsisWrapper[*synopsis.Flag]
	HelpText    string
	ManualText  string
	Description any
	Data        map[string]any
}

type commandDataCategory struct {
	Category        string
	VisibleCommands []*commandData
}

type flagDataList []*flagData
type flagDataCategory struct {
	Undocumented bool
	Category     string
	VisibleFlags flagDataList
}

var (
	// HelpTemplate provides the default help Go template that is rendered on the help
	// screen.  The preferred way to customize the help screen is to override its constituent
	// templates.  The template should otherwise define an entry point named "Help", which
	// you can use to define a from-scratch template.
	HelpTemplate = `
{{- define "Subcommands" -}}
{{ if or .CommandsByCategory .VisibleCommands -}}
{{ "\n" }}Sub-commands:{{ "\n" -}}
{{ range .CommandsByCategory -}}
{{ if and .VisibleCommands .Category }}{{ "\n" }}{{.Category}}:{{ "\n" }}{{ end -}}
{{ "\n" }}{{- template "SubcommandListing" .VisibleCommands -}}
{{ else }}
{{ "\n" }}{{- template "SubcommandListing" .VisibleCommands -}}
{{ end }}
{{ end -}}
{{- end -}}

{{- define "SubcommandListing" -}}
{{- range . -}}
{{ "\t" }}{{ .Names | BoldFirst | Join ", " }}{{ "\t" }}{{.HelpText}}{{ "\n" }}
{{- end -}}
{{- end -}}

{{- define "Flag" -}}
{{ "\t" }}{{ .Synopsis | print | ExtraSpaceBeforeFlag }}{{ "\t" }}{{.HelpText}}
{{- end -}}

{{- define "Flags" -}}
{{ range .FlagsByCategory -}}
{{ if and .VisibleFlags .Category }}{{ "\n" }}{{.Category}}:{{ "\n" }}{{ end -}}
{{ if .Undocumented -}}
{{ "\n" }}{{- template "InlineFlagListing" .VisibleFlags -}}
{{- else -}}
{{- template "FlagListing" .VisibleFlags -}}
{{- end -}}
{{- else -}}
{{- template "FlagListing" .VisibleFlags -}}
{{- end }}
{{- end -}}

{{- define "FlagListing" -}}
{{ if . }}{{ "\n" }}{{ end -}}
{{ range . }}
    {{- template "Flag" . -}}{{ "\n" }}
{{- end }}
{{- end -}}

{{- define "InlineFlagListing" -}}
{{ .Names | Join ", " | Wrap 4 }}
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
{{ "\n" }}Global options (specify before any sub-commands){{ "\n" }}
{{- template "Flags" .Persistent -}}
{{ end }}
{{- end -}}

{{/* Usage is the entry point, which calls flags, sub-commands */}} 
{{- define "Usage" -}}
usage: {{ if .SelectedCommand.Lineage -}}
	{{- .SelectedCommand.Lineage -}}
	{{- " " -}}
{{- end -}}
{{- .SelectedCommand.Synopsis | print | HangingIndent .SelectedCommand.HangingIndent -}}{{ "\n" }}

{{- if .SelectedCommand.Description -}}
{{ "\n" }}{{- .SelectedCommand.Description | Wrap 4 -}}
{{ else if .SelectedCommand.HelpText -}}
{{ "\n" }}{{- .SelectedCommand.HelpText | Wrap 4 -}}
{{- end -}}
{{- template "Flags" .SelectedCommand -}}
{{- template "Subcommands" .SelectedCommand -}}
{{- template "PersistentFlags" .SelectedCommand -}}
{{- template "ExtendedDescription" .SelectedCommand -}}
{{- end -}}

{{- template "Usage" $ -}}
`

	expressionTemplate = `
{{- define "Expression" -}}
{{ "\t" }}{{ .Synopsis }}{{ "\t" }}{{.HelpText}}
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
	VersionTemplate = "{{ .App.Name }}, version {{ .App.Version }}\n"

	// LicenseTemplate specifies the Go template for what is printed when
	//   the license flag or command is used.
	LicenseTemplate = "{{ .App.License | Wrap 4 }}\n"

	builtinFuncs = template.FuncMap{
		"Join": func(v string, args []string) string {
			return strings.Join(args, v)
		},
		"Repeat": func(count int, s string) string {
			return strings.Repeat(s, count)
		},
		"SpaceBefore": func(s string) string {
			if s == "" {
				return ""
			}
			return " " + s
		},
		"SpaceAfter": func(s string) string {
			if s == "" {
				return ""
			}
			return s + " "
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
)

func (c *commandData) withLineage(lineage string, persistent []*Flag) *commandData {
	c.Lineage = lineage
	c.Persistent = &persistentCommandData{
		VisibleFlags:    visibleFlags(persistent),
		FlagsByCategory: visibleFlagCategories(groupFlagsByCategory(persistent)),
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

func visibleFlagCategories(items flagsByCategory) []*flagDataCategory {
	res := make([]*flagDataCategory, 0, len(items))
	for _, a := range items {
		res = append(res, &flagDataCategory{
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

		visibleCategories = func(items commandsByCategory) []*commandDataCategory {
			res := make([]*commandDataCategory, 0, len(items))
			for _, a := range items {
				res = append(res, &commandDataCategory{
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
		Synopsis:           wrapSynopsis(val.newSynopsis()),
		VisibleArgs:        visibleArgs(val.VisibleArgs()),
		VisibleFlags:       visibleFlags(val.VisibleFlags()),
		VisibleCommands:    visibleCommands(val.VisibleSubcommands()),
		CommandsByCategory: visibleCategories(groupedByCategory(val.Subcommands)),
		FlagsByCategory:    visibleFlagCategories(groupFlagsByCategory(val.Flags)),
		Persistent: &persistentCommandData{
			VisibleFlags: []*flagData{},
		},
		Data: val.Data,
	}
}

func renderHelp(us *synopsis.Usage) string {
	sb := NewBuffer()
	us.HelpText(sb)
	return sb.String()
}

func flagAdapter(val *Flag) *flagData {
	syn := val.newSynopsis()
	return &flagData{
		Name:        val.Name,
		HelpText:    renderHelp(syn.Value.Usage),
		ManualText:  val.ManualText,
		Description: val.Description,
		Synopsis:    wrapSynopsis(syn),
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
