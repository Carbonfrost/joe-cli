package cli

type commandData struct {
	Name            string
	HelpText        string
	Synopsis        string
	VisibleCommands []*commandData
	VisibleFlags    []*flagData
	VisibleArgs     []*flagData
}

type flagData struct {
	Name     string
	Synopsis string
	HelpText string
}

var (
	HelpTemplate = `usage: {{.SelectedCommand.Synopsis}}
{{range .SelectedCommand.VisibleFlags}}
{{ "\t" }}{{.Synopsis}}{{ "\t" }}{{.HelpText}}{{end}}
{{range .SelectedCommand.VisibleCommands}}
{{ "\t" }}{{.Name}}{{ "\t" }}{{.HelpText}}{{end}}
`
	VersionTemplate = "{{ .App.Name }} {{ .App.Version }}"
)

func visibleFlags(items []*Flag) []*flagData {
	res := make([]*flagData, 0, len(items))
	for _, a := range items {
		if a.Hidden {
			continue
		}
		res = append(res, flagAdapter(a))
	}
	return res
}

func visibleArgs(items []*Arg) []*flagData {
	res := make([]*flagData, 0, len(items))
	for _, a := range items {
		if a.Hidden {
			continue
		}
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

func commandAdapter(val *Command) *commandData {
	return &commandData{
		Name:            val.Name,
		HelpText:        val.HelpText,
		Synopsis:        val.Synopsis(),
		VisibleArgs:     visibleArgs(val.Args),
		VisibleFlags:    visibleFlags(val.Flags),
		VisibleCommands: visibleCommands(val.Subcommands),
	}
}

func flagAdapter(val *Flag) *flagData {
	return &flagData{
		Name:     val.Name,
		HelpText: val.HelpText,
		Synopsis: val.Synopsis(),
	}
}

func argAdapter(val *Arg) *flagData {
	return &flagData{
		Name:     val.Name,
		HelpText: val.HelpText,
		Synopsis: val.Name,
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
