package cli

type commandData struct {
	Name               string
	Names              []string
	Description        string
	HelpText           string
	Synopsis           string
	VisibleCommands    []*commandData
	VisibleFlags       []*flagData
	VisibleArgs        []*flagData
	CommandsByCategory []*commandCategory
}

type flagData struct {
	Name     string
	Synopsis string
	HelpText string
}

type commandCategory struct {
	Category        string
	VisibleCommands []*commandData
}

var (
	HelpTemplate = `
{{- define "Subcommands" -}}
{{ range .CommandsByCategory }}
{{ if .Category }}{{.Category}}:{{ end }}
{{- range .VisibleCommands }}
{{ "\t" }}{{ .Names | Join ", " }}{{ "\t" }}{{.HelpText}}{{end}}
{{ else }}
{{- range .VisibleCommands }}
{{ "\t" }}{{.Name}}{{ "\t" }}{{.HelpText}}{{end}}
{{ end }}
{{- end -}}

{{- define "Flags" -}}
{{ range .VisibleFlags }}
{{ "\t" }}{{.Synopsis}}{{ "\t" }}{{.HelpText}}{{end}}
{{- end -}}

{{/* Usage is the entry point, which calls flags, subcommands */}} 
{{- define "Usage" -}}
usage: {{ if .CommandLineage }}{{.CommandLineage}} {{ end }}{{ .SelectedCommand.Synopsis }}
{{ if .SelectedCommand.Description }}
{{ .SelectedCommand.Description | Wrap 4 }}
{{- end -}}
{{- template "Flags" .SelectedCommand -}}
{{- template "Subcommands" .SelectedCommand -}}
{{- end -}}

{{- template "Usage" $ -}}
`

	// VersionTemplate specifies the Go template for what is printed when
	//   the version flag or command is used.
	VersionTemplate string = "{{ .App.Name }}, version {{ .App.Version }}\n"
)

func visibleFlags(items []*Flag) []*flagData {
	res := make([]*flagData, 0, len(items))
	for _, a := range items {
		res = append(res, flagAdapter(a))
	}
	return res
}

func visibleArgs(items []*Arg) []*flagData {
	res := make([]*flagData, 0, len(items))
	for _, a := range items {
		res = append(res, argAdapter(a))
	}
	return res
}

func visibleCommands(items []*Command) []*commandData {
	res := make([]*commandData, 0, len(items))
	for _, a := range items {
		res = append(res, commandAdapter(a))
	}
	return res
}

func visibleCategories(items CommandsByCategory) []*commandCategory {
	res := make([]*commandCategory, 0, len(items))
	for _, a := range items {
		res = append(res, &commandCategory{
			Category:        a.Category,
			VisibleCommands: visibleCommands(a.Commands),
		})
	}
	return res
}

func commandAdapter(val *Command) *commandData {
	return &commandData{
		Name:               val.Name,
		Names:              val.Names(),
		Description:        val.Description,
		HelpText:           val.HelpText,
		Synopsis:           val.Synopsis(),
		VisibleArgs:        visibleArgs(val.VisibleArgs()),
		VisibleFlags:       visibleFlags(val.VisibleFlags()),
		VisibleCommands:    visibleCommands(val.Subcommands),
		CommandsByCategory: visibleCategories(GroupedByCategory(val.Subcommands)),
	}
}

func flagAdapter(val *Flag) *flagData {
	syn := val.newSynopsis()
	return &flagData{
		Name:     val.Name,
		HelpText: syn.value.helpText,
		Synopsis: syn.formatString(false),
	}
}

func argAdapter(val *Arg) *flagData {
	return &flagData{
		Name:     val.Name,
		HelpText: val.HelpText,
		Synopsis: val.newSynopsis().formatString(),
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
