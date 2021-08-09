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
		"sub":    subcommandApp(),
	}
)

func fileRequiredApp() *cli.App {
	// <FILE>
	return &cli.App{
		Args: []*cli.Arg{
			{
				Name: "FILE",
			},
		},
	}
}

func optionalArgumentApp() *cli.App {
	// <arg>
	return &cli.App{
		Args: []*cli.Arg{
			{
				Name: "a",
			},
		},
	}
}

func subcommandApp() *cli.App {
	return &cli.App{
		Flags: []*cli.Flag{
			{
				Name: "flag1",
			},
		},
		Args: []*cli.Arg{
			{
				Name: "arg",
			},
		},
		Commands: []*cli.Command{
			{
				Name: "sub",
			},
		},
	}
}
