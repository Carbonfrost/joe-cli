package joe

import (
	"runtime/debug"

	. "github.com/Carbonfrost/joe-cli/extensions/template"
)

func newAppTemplate(data *generatorData) *Root {
	var deps []string
	if data.Dependencies.HTTP {
		deps = append(deps, "github.com/Carbonfrost/joe-cli-http")
	}
	var license Generator
	if data.License {
		license = File("cmd/{{ .App.Name }}/license.txt", Contents("No license is available with this build."))
	}

	return New(
		Data("App", data),
		GoGet(deps...),
		File("cmd/{{ .App.Name }}/main.go", Template(appGoTemplate), Gofmt()),
		license,
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
