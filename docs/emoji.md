## Emoji

> added in v0.3.0

A select set of emoji can be accessed by name using the `Emoji` template function.

* `{{ Emoji "Tada" }}` "🎉"
* `{{ Emoji "Fire" }}`  "🔥"
* `{{ Emoji "Sparkles" }}`  "✨"
* `{{ Emoji "Exclamation" }}`  "❗"
* `{{ Emoji "Bulb" }}` "💡"
* `{{ Emoji "X" }}` "❌"
* `{{ Emoji "HeavyCheckMark" }}` "✔️"
* `{{ Emoji "Warning" }}` "⚠️"
* `{{ Emoji "Play" }}`  "▶"

Here's an example using an emoji in a template:

```go
package main

import (
	"os"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/color"
)

const (
	updateTemplate = `{{ Emoji "Sparkles" }} A newer version of {{ .App }} is available (
{{- .CurrentVersion  }} -> {{ .NewVersion | Yellow }})`
)

func main() {
	(&cli.App{
		Uses: cli.Pipeline(
			color.Options{},
			cli.RegisterTemplate("Update", updateTemplate),
		),
		Action: cli.RenderTemplate("Update", func(*cli.Context) interface{} {
			return map[string]interface{}{
				"App":            "salsa",
				"NewVersion":     "0.5.0",
				"CurrentVersion": "0.4.2",
			}
		}),
	}).Run(os.Args)
}
```
