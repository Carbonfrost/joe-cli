package main

import (
    "os"

    cli "github.com/Carbonfrost/joe-cli"
{{ if .App.Extensions.Color -}}
    "github.com/Carbonfrost/joe-cli/extensions/color"
{{ end -}}
{{ if .App.Extensions.Table -}}
    "github.com/Carbonfrost/joe-cli/extensions/table"
{{ end -}}
)

{{- if .App.License }}
import (
    _ "embed"
)

    //go:embed license.txt
    var license string
{{ end -}}

func main() {
    createApp().Run(os.Args)
}

func createApp() *cli.App {
    return &cli.App{
        Name:     {{ .App.Name | Quote }},
        HelpText: {{ .App.HelpText | Quote }},
        Comment: {{ .App.Comment | Quote }},
{{- if .App.License }}
        License: license,
{{ end -}}
        Uses: cli.Pipeline(
{{- if .App.Extensions.Color }}
            &color.Options{},
{{- end -}}
{{- if .App.Extensions.Table }}
            &table.Options{},
{{- end -}}
        ),
        Action:  func (c *cli.Context) error {
            c.Stdout.WriteString("Hello, world!")
            return nil
        },
        Version: {{ .App.Version | Quote }},
        Args: []*cli.Arg{
{{- if .App.License }}
            {Uses: cli.PrintLicense() },
{{- end -}}
        },
    }
}
