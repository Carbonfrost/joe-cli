// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Logger", func() {

	Describe("New", func() {

		It("uses the empty name by default", func() {
			Expect(log.New().Name()).To(BeEmpty())
		})

		It("applies WithName", func() {
			Expect(log.New(log.WithName("audit")).Name()).To(Equal("audit"))
		})

		DescribeTable("options", func(opts []log.Option, expected types.GomegaMatcher) {
			var buf bytes.Buffer
			l := log.New(append(opts, log.WithOutput(&buf))...)
			l.Info("hello", "a", 1)

			Expect(buf.String()).To(expected)
		},
			Entry("text format by default", nil, ContainSubstring(`level=INFO msg=hello a=1`)),
			Entry(
				"WithLogFormat JSON",
				[]log.Option{log.WithLogFormat(log.JSONFormat)},
				ContainSubstring(`"level":"INFO","msg":"hello","a":1`),
			),
			Entry(
				"WithLevel excludes lower levels",
				[]log.Option{log.WithLevel(log.LevelError)},
				BeEmpty(),
			),
			Entry(
				"WithAddSource",
				[]log.Option{log.WithAddSource(true)},
				ContainSubstring(`source=`),
			),
		)

		It("applies options which are added after the logger is used", func() {
			var buf bytes.Buffer
			l := log.New(log.WithOutput(&buf))
			l.Info("first")
			l.Apply(log.WithLogFormat(log.JSONFormat))
			l.Info("second")

			Expect(buf.String()).To(ContainSubstring("msg=first"))
			Expect(buf.String()).To(ContainSubstring(`"msg":"second"`))
		})
	})

	Describe("AddSource", func() {

		It("reports the position of the caller of the log function", func() {
			var buf bytes.Buffer
			l := log.New(log.WithOutput(&buf), log.WithAddSource(true), log.WithLogFormat(log.JSONFormat))
			l.Info("hello")

			var actual struct {
				Source struct {
					File string `json:"file"`
				} `json:"source"`
			}
			Expect(json.Unmarshal(buf.Bytes(), &actual)).NotTo(HaveOccurred())
			Expect(actual.Source.File).To(HaveSuffix("log_test.go"))
		})
	})

	Describe("default action", func() {

		It("registers the logger in the context by name", func() {
			var actual *log.Logger
			app := &cli.App{
				Uses: log.New(log.WithName("audit")),
				Action: func(ctx context.Context) {
					actual = log.FromContext(ctx, "audit")
				},
			}
			Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(actual).NotTo(BeNil())
			Expect(actual.Name()).To(Equal("audit"))
		})

		It("registers the logger with the context services", func() {
			var actual bool
			app := &cli.App{
				Uses: log.New(),
				Action: func(ctx context.Context) {
					_, actual = log.Services(ctx).Lookup("")
				},
			}
			Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(actual).To(BeTrue())
		})

		It("uses the Stderr of the app by default", func() {
			var buf bytes.Buffer
			app := &cli.App{
				Stderr: &buf,
				Uses:   log.New(),
				Action: func(ctx context.Context) {
					log.InfoContext(ctx, "hello")
				},
			}
			Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("msg=hello"))
		})

		It("adds the flags for the default logger", func() {
			var actual []string
			app := &cli.App{
				Uses: log.New(),
				Action: func(c *cli.Context) {
					for _, f := range c.Command().Flags {
						actual = append(actual, f.Name)
					}
				},
			}
			Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(actual).To(ContainElements("log-level", "log-format", "log-add-source"))
		})

		It("doesn't add flags for a named logger", func() {
			var actual []string
			app := &cli.App{
				Uses: log.New(log.WithName("audit")),
				Action: func(c *cli.Context) {
					for _, f := range c.Command().Flags {
						actual = append(actual, f.Name)
					}
				},
			}
			Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(actual).NotTo(ContainElement(HavePrefix("log-")))
		})
	})

	Describe("flags", func() {

		DescribeTable("examples", func(args string, expected types.GomegaMatcher) {
			var buf bytes.Buffer
			app := &cli.App{
				Stderr: &buf,
				Uses:   log.New(),
				Action: func(ctx context.Context) {
					log.DebugContext(ctx, "hello")
					log.InfoContext(ctx, "hello")
				},
			}
			args = "app " + args
			Expect(app.RunContext(context.Background(), strings.Fields(args))).NotTo(HaveOccurred())
			Expect(buf.String()).To(expected)
		},
			Entry("log-level", "--log-level=debug", ContainSubstring("level=DEBUG")),
			Entry("log-level excludes", "--log-level=error", BeEmpty()),
			Entry("log-format", "--log-format=json", ContainSubstring(`"msg":"hello"`)),
			Entry("log-add-source", "--log-add-source", ContainSubstring("source=")),
		)

		It("names the flags of a named logger after it", func() {
			var buf bytes.Buffer
			app := &cli.App{
				Stderr: &buf,
				Uses:   log.New(log.WithName("audit")),
				Flags: []*cli.Flag{
					{Uses: log.SetLogFormat("audit")},
				},
				Action: func(ctx context.Context) {
					log.FromContext(ctx, "audit").Info("hello")
				},
			}
			err := app.RunContext(context.Background(), []string{"app", "--audit-format=json"})
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring(`"msg":"hello"`))
		})

		It("reports an error for an unknown format", func() {
			app := &cli.App{
				Uses:   log.New(),
				Action: func() {},
			}
			err := app.RunContext(context.Background(), []string{"app", "--log-format=xml"})
			Expect(err).To(MatchError(ContainSubstring(`unexpected log format "xml"`)))
		})
	})

	Describe("log functions", func() {

		It("delegate to the default logger of the app", func() {
			var buf bytes.Buffer
			app := &cli.App{
				Stderr: &buf,
				Uses:   log.New(),
				Action: func() {
					// No context: this bridges to the current app
					log.Info("hello")
				},
			}
			Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("msg=hello"))
		})

		It("delegate to the default logger in the context", func() {
			var buf bytes.Buffer
			app := &cli.App{
				Uses: log.New(log.WithOutput(&buf)),
				Action: func(ctx context.Context) {
					log.InfoContext(ctx, "hello")
				},
			}
			Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("msg=hello"))
		})

		It("fall back to the slog default when there is no app", func() {
			var buf bytes.Buffer
			restore := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
			defer slog.SetDefault(restore)

			log.Info("hello")
			Expect(buf.String()).To(ContainSubstring("msg=hello"))
		})
	})

	Describe("ContextServices", func() {

		It("is unique per app", func() {
			var actual1, actual2 [2]bool
			app1 := &cli.App{
				Uses: log.New(log.WithName("0")),
				Action: func(ctx context.Context) {
					_, actual1[0] = log.Services(ctx).Lookup("0")
					_, actual1[1] = log.Services(ctx).Lookup("1")
				},
			}
			app2 := &cli.App{
				Uses: log.New(log.WithName("1")),
				Action: func(ctx context.Context) {
					_, actual2[0] = log.Services(ctx).Lookup("0")
					_, actual2[1] = log.Services(ctx).Lookup("1")
				},
			}

			Expect(app1.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
			Expect(app2.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())

			Expect(actual1).To(Equal([2]bool{true, false}))
			Expect(actual2).To(Equal([2]bool{false, true}))
		})
	})
})

var _ = Describe("LogFormat", func() {

	DescribeTable("String", func(f log.LogFormat, expected string) {
		Expect(f.String()).To(Equal(expected))
	},
		Entry("text", log.TextFormat, "text"),
		Entry("JSON", log.JSONFormat, "json"),
	)

	DescribeTable("UnmarshalText", func(text string, expected log.LogFormat) {
		var actual log.LogFormat
		Expect(actual.UnmarshalText([]byte(text))).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	},
		Entry("text", "text", log.TextFormat),
		Entry("JSON", "json", log.JSONFormat),
		Entry("trims space", " json ", log.JSONFormat),
	)

	It("reports an error for an unknown name", func() {
		var actual log.LogFormat
		Expect(actual.UnmarshalText([]byte("xml"))).To(MatchError(`unexpected log format "xml"`))
	})
})
