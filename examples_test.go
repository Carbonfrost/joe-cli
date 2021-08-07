package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
)

func app(usage string) *cli.App {
	return appExamples[usage]
}

var (
	appExamples = map[string]*cli.App{
		"<FILE>": fileRequiredApp(),
		"<arg>":  optionalArgumentApp(),
	}
)

func fileRequiredApp() *cli.App {
	// <FILE>
	return &cli.App{
		Args: []cli.Arg{
			&cli.StringArg{
				Name: "FILE",
			},
		},
	}
}

func optionalArgumentApp() *cli.App {
	// <arg>
	return &cli.App{
		Args: []cli.Arg{
			&cli.StringArg{
				Name: "a",
			},
		},
	}
}
