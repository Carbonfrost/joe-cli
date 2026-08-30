// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/clitest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHelpScreen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Help Screen Suite")
}

// HelpScreen labels the specs which compare the rendered help screen against
// a golden file.  Use `ginkgo --label-filter=help-screen` to run only these, or
// `--label-filter='!help-screen'` to skip them when the layout is in flux.
var HelpScreen = Label("help-screen")

var _ = Describe("help screen", HelpScreen, func() {
	DescribeTable("layout",
		func(fixture string, app *cli.App) {
			expectGoldenFile(fixture, renderScreen(app, "app --help"))
		},

		Entry("help topic listing", "help_topic_listing", &cli.App{
			Name: "app",
			Commands: []*cli.Command{
				{Name: "init", HelpText: "Initialize the workspace"},
			},
			Uses: cli.Pipeline(
				cli.HelpTopic{Name: "credentials", HelpText: "Providing usernames and passwords"},
				cli.HelpTopic{Name: "everyday", HelpText: "A useful minimum set of commands for everyday use"},
				cli.HelpTopic{Name: "faq", HelpText: "Frequently asked questions about using the app"},
				cli.HelpTopic{Name: "glossary", HelpText: "A glossary of terminology"},
				cli.HelpTopic{Name: "migration", HelpText: "How to migrate from other tools"},
				cli.HelpTopic{Name: "namespaces", HelpText: "How to use namespaces"},
				cli.HelpTopic{Name: "tutorial", HelpText: "A tutorial for new users"},
				cli.HelpTopic{Name: "workflows", HelpText: "An overview of recommended workflows with the app"},
			),
		}),
		Entry("flags only", "flags_only", &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					// Plainest configuration: name only, implicitly a string
					Name: "normal",
				},
				{
					Name:     "config",
					HelpText: "Loads configuration from {FILE}s",
				},
				{
					// Long and short name; the synopsis shows both
					Name:     "verbose",
					Aliases:  []string{"v"},
					Value:    cli.Bool(),
					HelpText: "Display verbose output",
				},
				{
					Name:     "count",
					Value:    cli.Int(),
					HelpText: "Stop after {COUNT} items",
				},
				{
					// UsageText overrides the synopsis generated from the value type
					Name:      "output",
					UsageText: "<destination>",
					HelpText:  "Write results to destination",
				},
				{
					// Optional makes the value itself optional
					Name:     "level",
					Value:    cli.Int(),
					Options:  cli.Optional,
					HelpText: "Set the verbosity level",
				},
				{
					// Repeatable list value
					Name:     "tag",
					Value:    cli.List(),
					HelpText: "Add {TAG} to the output",
				},
				{
					// Short name only
					Name:     "n",
					Value:    cli.Bool(),
					HelpText: "Perform a dry run",
				},
				{
					// Hidden flags are absent from the screen entirely
					Name:    "hidden",
					Options: cli.Hidden,
				},
			},
		}),
	)
})

// expectGoldenFile compares the rendered screen byte for byte with the contents
// of testdata/help_screen/<fixture>.txt.  Set JOE_UPDATE_GOLDEN=1 to rewrite the
// fixtures from the current output, then review the diff before committing.
func expectGoldenFile(fixture, actual string) {
	GinkgoHelper()

	actual = normalizeLineEndings(actual)

	// Guard against a fixture which captures a screen that never rendered
	Expect(actual).NotTo(BeEmpty())

	name := filepath.Join("testdata", "help_screen", fixture+".txt")
	if os.Getenv("JOE_UPDATE_GOLDEN") != "" {
		Expect(os.MkdirAll(filepath.Dir(name), 0o755)).NotTo(HaveOccurred())
		Expect(os.WriteFile(name, []byte(actual), 0o644)).NotTo(HaveOccurred())
	}

	expected, err := os.ReadFile(name)
	expected = normalizeLineEndings(expected)

	Expect(err).NotTo(HaveOccurred(), "missing fixture %s; re-run with JOE_UPDATE_GOLDEN=1", name)
	Expect(actual).To(Equal(string(expected)))
}

func normalizeLineEndings[T ~string | ~[]byte](src T) T {
	return T(bytes.ReplaceAll([]byte(src), []byte("\r\n"), []byte("\n")))
}

func disableConsoleColor() func() {
	os.Setenv("TERM", "dumb")
	return func() {
		os.Setenv("TERM", "0")
	}
}

func renderScreen(app *cli.App, args string) string {
	defer disableConsoleColor()()

	arguments, _ := cli.Split(args)
	buffer, _ := clitest.Command(app, arguments...).CombinedOutput()
	return string(buffer)
}
