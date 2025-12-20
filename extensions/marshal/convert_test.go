package marshal_test

import (
	"encoding/json"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

var _ = Describe("From", func() {

	DescribeTableSubtree("examples", func(target any, expected types.GomegaMatcher, expectedJSON any) {

		It("converts marshal", func() {
			Expect(marshal.From(target)).To(expected)
		})

		It("converts to marshal JSON", func() {
			data, err := json.Marshal(marshal.From(target))
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(MatchJSON(expectedJSON))
		})
	},

		Entry("App", &cli.App{
			Name:      "Name",
			Version:   "Version",
			BuildDate: time.Date(2023, 3, 3, 0, 0, 0, 0, time.UTC),
			Author:    "Author",
			Copyright: "Copyright",
			Comment:   "Comment",
			Commands: []*cli.Command{
				{Name: "Command"},
			},
			Flags: []*cli.Flag{
				{Name: "Flag"},
			},
			Args: []*cli.Arg{
				{Name: "Arg"},
			},
			Data:       map[string]any{"key": "value"},
			Options:    cli.Hidden,
			HelpText:   "HelpText",
			ManualText: "ManualText",
			UsageText:  "UsageText",
			License:    "License",
		}, MatchFields(IgnoreUnexportedExtras, Fields{
			"Name": Equal("Name"),
			"Commands": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": BeAssignableToTypeOf(marshal.Command{}),
			}),
			"Flags": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": BeAssignableToTypeOf(marshal.Flag{}),
			}),
			"Args": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": BeAssignableToTypeOf(marshal.Arg{}),
			}),
			"Comment":    Equal("Comment"),
			"Data":       Equal(map[string]any{"key": "value"}),
			"Options":    Equal(cli.Hidden),
			"HelpText":   Equal("HelpText"),
			"ManualText": Equal("ManualText"),
			"UsageText":  Equal("UsageText"),
			"Version":    Equal("Version"),
			"Author":     Equal("Author"),
			"License":    Equal("License"),
			"Copyright":  Equal("Copyright"),
			"BuildDate":  Equal(time.Date(2023, 3, 3, 0, 0, 0, 0, time.UTC)),
		}), `{
               "name": "Name",
               "commands": [
                 {
                   "name": "Command"
                 }
               ],
               "flags": [
                 {
                   "name": "Flag"
                 }
               ],
               "args": [
                 {
                   "name": "Arg"
                 }
               ],
               "helpText": "HelpText",
               "manualText": "ManualText",
               "usageText": "UsageText",
               "version": "Version",
               "buildDate": "2023-03-03T00:00:00Z",
               "author": "Author",
               "copyright": "Copyright",
               "license": "License",
               "comment": "Comment",
               "options": "HIDDEN",
               "data": {
                 "key": "value"
               }
             }
       `),

		Entry("Command", &cli.Command{
			Name: "Name",
			Subcommands: []*cli.Command{
				{Name: "Command"},
			},
			Flags: []*cli.Flag{
				{Name: "Flag"},
			},
			Args: []*cli.Arg{
				{Name: "Arg"},
			},
			Aliases:    []string{"c"},
			Category:   "Category",
			Comment:    "Comment",
			Data:       map[string]any{"key": "value"},
			Options:    cli.Hidden,
			HelpText:   "HelpText",
			ManualText: "ManualText",
			UsageText:  "UsageText",
		}, MatchFields(IgnoreUnexportedExtras, Fields{
			"Name": Equal("Name"),
			"Subcommands": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": BeAssignableToTypeOf(marshal.Command{}),
			}),
			"Flags": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": BeAssignableToTypeOf(marshal.Flag{}),
			}),
			"Args": MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": BeAssignableToTypeOf(marshal.Arg{}),
			}),
			"Aliases":    Equal([]string{"c"}),
			"Category":   Equal("Category"),
			"Comment":    Equal("Comment"),
			"Data":       Equal(map[string]any{"key": "value"}),
			"Options":    Equal(cli.Hidden),
			"HelpText":   Equal("HelpText"),
			"ManualText": Equal("ManualText"),
			"UsageText":  Equal("UsageText"),
		}), `{
               "name": "Name",
               "subcommands": [
                 {
                   "name": "Command"
                 }
               ],
               "flags": [
                 {
                   "name": "Flag"
                 }
               ],
               "args": [
                 {
                   "name": "Arg"
                 }
               ],
               "aliases": [
                 "c"
               ],
               "category": "Category",
               "comment": "Comment",
               "data": {
                 "key": "value"
               },
               "options": "HIDDEN",
               "helpText": "HelpText",
               "manualText": "ManualText",
               "usageText": "UsageText"
             }`),

		Entry("Flag", &cli.Flag{
			Name:        "Flag",
			Aliases:     []string{"f"},
			EnvVars:     []string{"ENV_VAR"},
			FilePath:    "/usr",
			HelpText:    "HelpText",
			ManualText:  "ManualText",
			Category:    "Category",
			UsageText:   "UsageText",
			DefaultText: "DefaultText",
			Options:     cli.Hidden,
			Data:        map[string]any{"k": "v"},
		}, MatchFields(IgnoreUnexportedExtras, Fields{
			"Name":        Equal("Flag"),
			"Aliases":     Equal([]string{"f"}),
			"EnvVars":     Equal([]string{"ENV_VAR"}),
			"FilePath":    Equal("/usr"),
			"HelpText":    Equal("HelpText"),
			"ManualText":  Equal("ManualText"),
			"Category":    Equal("Category"),
			"UsageText":   Equal("UsageText"),
			"DefaultText": Equal("DefaultText"),
			"Options":     Equal(cli.Hidden),
			"Data":        Equal(map[string]any{"k": "v"}),
		}),
			`{
               "name": "Flag",
               "aliases": [
                 "f"
               ],
               "envVars": [
                 "ENV_VAR"
               ],
               "filePath": "/usr",
               "helpText": "HelpText",
               "manualText": "ManualText",
               "category": "Category",
               "usageText": "UsageText",
               "defaultText": "DefaultText",
               "options": "HIDDEN",
               "data": {
                 "k": "v"
               }
             }`),

		Entry("Arg", &cli.Arg{
			Name:        "Arg",
			EnvVars:     []string{"ENV_VAR"},
			FilePath:    "/usr",
			HelpText:    "HelpText",
			ManualText:  "ManualText",
			Category:    "Category",
			UsageText:   "UsageText",
			DefaultText: "DefaultText",
			Options:     cli.Hidden,
			Data:        map[string]any{"f": "b"},
		}, MatchFields(IgnoreUnexportedExtras, Fields{
			"Name":        Equal("Arg"),
			"EnvVars":     Equal([]string{"ENV_VAR"}),
			"FilePath":    Equal("/usr"),
			"HelpText":    Equal("HelpText"),
			"ManualText":  Equal("ManualText"),
			"Category":    Equal("Category"),
			"UsageText":   Equal("UsageText"),
			"DefaultText": Equal("DefaultText"),
			"Options":     Equal(cli.Hidden),
			"Data":        Equal(map[string]any{"f": "b"}),
		}),
			`{
               "name": "Arg",
               "envVars": [
                 "ENV_VAR"
               ],
               "filePath": "/usr",
               "helpText": "HelpText",
               "manualText": "ManualText",
               "category": "Category",
               "usageText": "UsageText",
               "defaultText": "DefaultText",
               "options": "HIDDEN",
               "data": {
                 "f": "b"
               }
             }`),
	)

})
