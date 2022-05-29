package joe

import (
	"runtime/debug"

	. "github.com/Carbonfrost/joe-cli/extensions/template"
)

func newAppTemplate(data *generatorData) *Root {
	var deps []string
	if data.Dependencies.Http {
		deps = append(deps, "github.com/Carbonfrost/joe-cli-http")
	}
	return New(
		Data("App", data),
		GoGet(deps...),
		File("cmd/{{ .App.Name }}/main.go", Template(appGoTemplate), Gofmt()),
	)
}

func newInitTemplate() *Root {
	return New(
		GoGet("github.com/Carbonfrost/joe-cli" + moduleVersion()),
	)
}

func moduleVersion() string {
	if b, ok := debug.ReadBuildInfo(); ok && b.Main.Version != "" {
		return "@" + b.Main.Version
	}
	return ""
}
