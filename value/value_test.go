// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value_test

import (
	"bytes"
	"context"
	"math"
	"testing/fstest"

	cli "github.com/Carbonfrost/joe-cli"
	joeclifakes "github.com/Carbonfrost/joe-cli/joe-clifakes"
	"github.com/Carbonfrost/joe-cli/value"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Value", func() {

	Describe("Set", func() {

		DescribeTable("generic values",
			func(f *cli.Flag, arguments string, expected types.GomegaMatcher) {
				act := new(joeclifakes.FakeAction)
				app := &cli.App{
					Name: "app",
					Flags: []*cli.Flag{
						f,
					},
					Action: act,
				}

				args, _ := cli.Split(arguments)
				err := app.RunContext(context.Background(), args)
				Expect(err).NotTo(HaveOccurred())
				captured := cli.FromContext(cli.FromContext(act.ExecuteArgsForCall(0)))
				Expect(captured.Value("o")).To(expected)
			},

			Entry(
				"ByteLength (via text unmarshaler)",
				&cli.Flag{Name: "o", Value: new(value.ByteLength)},
				"app -o 127GB",
				Equal(value.ByteLength(127*1000*1000*1000)),
			),
			Entry(
				"Hex (via text unmarshaler)",
				&cli.Flag{Name: "o", Value: new(value.Hex)},
				"app -o A75E",
				Equal(value.Hex(0xA75E)),
			),
			Entry(
				"Octal (via text unmarshaler)",
				&cli.Flag{Name: "o", Value: new(value.Octal)},
				"app -o 127",
				Equal(value.Octal(0o127)),
			),
		)
	})
})

var _ = Describe("JSON", func() {

	Describe("Set", func() {
		It("delegates to the internal value", func() {
			var data []byte
			actual := value.JSON(&data)
			err := actual.Set("CECE")
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal([]byte{0xCE, 0xCE}))
		})

		It("returns an error for values that can't process JSON", func() {
			var data struct{}
			actual := value.JSON(&data)

			err := actual.Set("OK")
			Expect(err).To(MatchError("can't set value directly; must read from file"))
		})
	})

	Describe("String", func() {
		It("delegates to the internal value", func() {
			data := "cli"
			actual := value.JSON(&data).String()
			Expect(actual).To(Equal("cli"))
		})

		It("returns empty string for values that can't process JSON", func() {
			var data struct{}
			actual := value.JSON(&data).String()
			Expect(actual).To(BeEmpty())
		})
	})

	Describe("SetData", func() {

		type pogo struct {
			J string `json:"j"`
			K int    `json:"k"`
		}

		DescribeTable("examples", func(data string, expectedErr, expected types.GomegaMatcher) {
			var actual pogo
			value := value.JSON(&actual)
			err := cli.SetData(value, bytes.NewReader([]byte(data)))
			Expect(err).To(expectedErr)
			Expect(actual).To(expected)
			Expect(&actual).To(Equal(value.Get()))
		},
			Entry(
				"nominal",
				`{"j": "first", "k": 20}`,
				Not(HaveOccurred()),
				Equal(pogo{J: "first", K: 20}),
			),
			Entry(
				"nominal",
				`"invalid JSON"`,
				HaveOccurred(),
				BeZero(),
			),
			Entry(
				"empty string is ignored",
				``,
				Not(HaveOccurred()),
				BeZero(),
			),
		)
	})

	It("supports copying via EachOccurrence", func() {
		var seen []*cli.NameValue

		seenWith := func(nv *cli.NameValue) error {
			c := *nv // Must copy the value so we have each instance that occurred
			seen = append(seen, &c)
			return nil
		}

		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:    "f",
					Value:   value.JSON(new(cli.NameValue)),
					Options: cli.EachOccurrence | cli.FileReference,
					Action: func(c *cli.Context) {
						seenWith(c.NameValue(""))
					},
				},
			},
			FS: fstest.MapFS{
				"h.json": {Data: []byte(`{ "Name": "H", "Value": "0" }`)},
				"i.json": {Data: []byte(`{ "Name": "I", "Value": "1" }`)},
			},
			Action: new(joeclifakes.FakeAction),
		}

		args, _ := cli.Split("app -f h.json -f i.json")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())

		Expect(seen).To(ContainElements(
			&cli.NameValue{
				Name:  "H",
				Value: "0",
			}, &cli.NameValue{
				Name:  "I",
				Value: "1",
			}))
	})

})

var _ = Describe("ByteLength", func() {

	Describe("ParseByteLength", func() {

		DescribeTable("examples", func(text string, expected int) {
			actual, err := value.ParseByteLength(text)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(int(expected)))
		},
			Entry("bare value", "69", 69),
			Entry("space between", "829 B", 829),
			Entry("magnitude B", "1B", 1),
			Entry("magnitude kB", "1kB", 1000),
			Entry("magnitude MB", "1MB", int(math.Pow(1000, 2))),
			Entry("magnitude GB", "1GB", int(math.Pow(1000, 3))),
			Entry("magnitude TB", "1TB", int(math.Pow(1000, 4))),
			Entry("magnitude PB", "1PB", int(math.Pow(1000, 5))),
			Entry("magnitude EB", "1EB", int(math.Pow(1000, 6))),
			Entry("magnitude ZB", "1ZB", int(math.Pow(1000, 7))),
			Entry("magnitude YB", "1YB", int(math.Pow(1000, 8))),
			Entry("magnitude RB", "1RB", int(math.Pow(1000, 9))),
			Entry("magnitude QB", "1QB", int(math.Pow(1000, 10))),
			Entry("magnitude KiB", "1KiB", 1024),
			Entry("magnitude MiB", "1MiB", int(math.Pow(1024, 2))),
			Entry("magnitude GiB", "1GiB", int(math.Pow(1024, 3))),
			Entry("magnitude TiB", "1TiB", int(math.Pow(1024, 4))),
			Entry("magnitude PiB", "1PiB", int(math.Pow(1024, 5))),
			Entry("magnitude EiB", "1EiB", int(math.Pow(1024, 6))),
			Entry("magnitude ZiB", "1ZiB", int(math.Pow(1024, 7))),
			Entry("magnitude YiB", "1YiB", int(math.Pow(1024, 8))),
		// IEC 80000 doesn't (yet) define RiB or QiB)
		)

		DescribeTable("errors", func(text string, expected string) {
			_, err := value.ParseByteLength(text)
			Expect(err).To(MatchError(ContainSubstring(expected)))
		},
			Entry("fractional bytes", "69.4", "invalid syntax"),
			Entry("fractional bytes space between", "829.3 B", "invalid syntax"),
		)

	})
})

var _ = Describe("Octal", func() {
	var _ = Describe("UnmarshalText", func() {
		DescribeTable("examples", func(text string, expected value.Octal) {
			octal := new(value.Octal)
			err := octal.UnmarshalText([]byte(text))
			Expect(err).NotTo(HaveOccurred())
			Expect(*octal).To(Equal(expected))
		},
			Entry("nominal", "70", value.Octal(0o70)),
			Entry("0o prefix", "0o40", value.Octal(0o40)),
		)

		DescribeTable("errors", func(text string, expected types.GomegaMatcher) {
			err := new(value.Octal).UnmarshalText([]byte(text))
			Expect(err).To(expected)
		},
			Entry("invalid chars", "GZ", MatchError("not a valid number: GZ")),
			Entry("empty string", "", MatchError(`empty string is not a valid number`)),
		)
	})

	Describe("String", func() {
		It("formats as a string", func() {
			Expect(value.Octal(0o110).String()).To(Equal("0o110"))
		})
	})
})

var _ = Describe("Hex", func() {
	var _ = Describe("UnmarshalText", func() {
		DescribeTable("examples", func(text string, expected value.Hex) {
			hex := new(value.Hex)
			err := hex.UnmarshalText([]byte(text))
			Expect(err).NotTo(HaveOccurred())
			Expect(*hex).To(Equal(expected))
		},
			Entry("nominal", "80", value.Hex(0x80)),
			Entry("0x prefix", "0x40", value.Hex(0x40)),
		)

		DescribeTable("errors", func(text string, expected types.GomegaMatcher) {
			err := new(value.Hex).UnmarshalText([]byte(text))
			Expect(err).To(expected)
		},
			Entry("invalid chars", "GZ", MatchError("not a valid number: GZ")),
			Entry("empty string", "", MatchError("empty string is not a valid number")),
		)
	})

	Describe("String", func() {
		It("formats as a string", func() {
			Expect(value.Hex(0x5BE).String()).To(Equal("0x5BE"))
		})
	})

})
