package cli

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

var (
	shellUnsafeChars = regexp.MustCompile(`\W*`)
)

// CompletionType enumerates the supported kinds of completions
type CompletionType int

// Completer is the shell auto-complete function for the flag, arg, or value
type Completion interface {
	Complete(*CompletionContext) []CompletionItem
}

// StandardCompletion enumerates standard completion results
type StandardCompletion int

// CompletionContext provides information about the completion request
type CompletionContext struct {
	// Context is the context where completion is occurring
	Context *Context

	// Args that have been passed to the command so far
	Args []string

	// Incomplete the token that is incomplete being completed
	Incomplete string

	// Bindings gets the bindings that were partially parsed
	Bindings BindingMap

	// Err gets the error that occurred during parsing, likely *ParserError
	Err error
}

// CompletionFunc provide a function that can be used as a Completer
type CompletionFunc func(*CompletionContext) []CompletionItem

const (
	TokenCompletionType     CompletionType = 0
	FileCompletionType      CompletionType = 1
	DirectoryCompletionType CompletionType = 2
)

const (
	FileCompletion StandardCompletion = iota
	DirectoryCompletion
)

// CompletionItem defines an item displayed in the completion results
type CompletionItem struct {
	Type     CompletionType
	Value    string
	HelpText string
}

// ShellComplete provides the implementation of the shell-specific
// completion handler
type ShellComplete interface {
	GetCompletionRequest() (args []string, incomplete string)
	SetOptions(map[string]string)
	FormatCompletions([]CompletionItem) string
	GetSourceTemplate() *Template
}

type completionData struct {
	App struct {
		Name string
	}
	CompletionFunc   string
	JoeCompletionVar string
	Shell            string
	ShellComplete    ShellComplete
}

type zshComplete struct {
	noDesc bool
}

const (
	robustParseModeEnabledKey = "__RobustParseModeEnabled"
	shellCompletesKey         = "__ShellCompletes"
	robustParseResultKey      = "__RobustParseResult"

	zshSourceScript = `

#compdef {{ .App.Name }}
{{ .CompletionFunc }}() {
    local -a completions
    local -a completions_with_descriptions
    local -a response

    (( ! $+commands[{{ .App.Name }}] )) && return 1

    response=("${(@f)$(env COMP_WORDS="${words[*]}" COMP_CWORD=$((CURRENT-1)) \
{{ .JoeCompletionVar }}=zsh {{ .App.Name }})}")
    for type key descr in ${response}; do
        if [[ "$type" == "plain" ]]; then
            if [[ "$descr" == "_" ]]; then
                completions+=("$key")
            else
                completions_with_descriptions+=("$key":"$descr")
            fi
        elif [[ "$type" == "dir" ]]; then
            _path_files -/
        elif [[ "$type" == "file" ]]; then
            _path_files -f
        fi
    done
    if [ -n "$completions_with_descriptions" ]; then
        _describe -V unsorted completions_with_descriptions -U
    fi
    if [ -n "$completions" ]; then
        compadd -U -V unsorted -a completions
    fi
}
compdef {{ .CompletionFunc }} {{ .App.Name }};
`
)

func newCompletionData(c *Context) *completionData {
	appName := c.App().Name
	slug := strings.ReplaceAll(shellUnsafeChars.ReplaceAllString(appName, ""), "-", "_")
	envVar := fmt.Sprintf("_JOE_%s_COMPLETE", strings.ToUpper(slug))
	shell := os.Getenv(envVar)

	return &completionData{
		App: struct{ Name string }{
			Name: appName,
		},
		Shell:            shell,
		ShellComplete:    c.shellCompletes()[shell],
		CompletionFunc:   fmt.Sprintf("_%s_completion_func", slug),
		JoeCompletionVar: envVar,
	}
}

// ShellCompleteIntegration provides an action which renders the particular
// shell complete integration
func ShellCompleteIntegration(name string, s ShellComplete) Action {
	return Pipeline(
		func(c *Context) {
			c.shellCompletes()[name] = s
		},
		&Prototype{
			Name:    name + "-complete",
			Options: Hidden | Exits,
			Value:   new(bool),
			Setup: Setup{
				Action: func(c *Context) error {
					data := newCompletionData(c)
					tpl := s.GetSourceTemplate()
					return tpl.Execute(c.Stdout, data)
				},
			},
		})
}

// ApplyShellCompletion detects whether a dynamic shell completion request has been added to
// the environment and activates the corresponding supported response.
func ApplyShellCompletion() Action {
	return Setup{
		Uses: func(c *Context) {
			cc := newCompletionData(c)
			if cc.Shell != "" {
				setupRobustParsingMode(c)
			}
		},
	}
}

func setupCompletion(c *Context) error {
	return c.Do(AddFlags([]*Flag{
		{Name: "zsh-completion", Uses: ShellCompleteIntegration("zsh", &zshComplete{})},
	}...),
		ApplyShellCompletion(),
	)
	return nil
}

func setupRobustParsingMode(c *Context) {
	// Activate robust parsing which causes errors in parsing to be
	// ignored so that we have an incomplete binding
	c.SetData(robustParseModeEnabledKey, true)
}

// Complete considers the given arguments and completion request to determine
// completion items
func (c *Context) Complete(args []string, incomplete string) []CompletionItem {
	setupRobustParsingMode(c)

	_ = c.Execute(args)
	re := c.robustParseResult()
	return c.complete(args, incomplete, re)
}

func (c *Context) complete(args []string, incomplete string, re *robustParseResult) []CompletionItem {
	cc := &CompletionContext{
		Context:    c,
		Args:       args,
		Incomplete: incomplete,
		Bindings:   re.bindings,
		Err:        re.err,
	}
	return c.target().completion().Complete(cc)
}

func (c *Context) robustParsingMode() bool {
	_, ok := c.LookupData(robustParseModeEnabledKey)
	return ok
}

func (c *Context) robustParseResult() *robustParseResult {
	var re *robustParseResult
	if e, ok := c.LookupData(robustParseResultKey); ok {
		re, _ = e.(*robustParseResult)
	}
	return re
}

func (c *Context) setRobustParseResult(r *robustParseResult) {
	c.SetData(robustParseResultKey, r)
}

func (c *Context) shellCompletes() map[string]ShellComplete {
	l, ok := c.LookupData(shellCompletesKey)
	if !ok {
		l = map[string]ShellComplete{}
		c.SetData(shellCompletesKey, l)
	}
	return l.(map[string]ShellComplete)
}

func (f CompletionFunc) Complete(c *CompletionContext) []CompletionItem {
	if f == nil {
		return nil
	}
	return f(c)
}

func (s StandardCompletion) Complete(c *CompletionContext) []CompletionItem {
	switch s {
	case FileCompletion:
		return []CompletionItem{
			{Value: c.Incomplete, Type: FileCompletionType},
		}
	case DirectoryCompletion:
		return []CompletionItem{
			{Value: c.Incomplete, Type: DirectoryCompletionType},
		}
	}
	panic(fmt.Sprintf("unexpected value: %v", s))
}

func (*zshComplete) GetCompletionRequest() (args []string, incomplete string) {
	cwords, _ := Split(os.Getenv("COMP_WORDS"))
	cword, _ := strconv.Atoi(os.Getenv("COMP_CWORD"))

	if cword < len(cwords) {
		args = cwords[1:cword]
		incomplete = cwords[cword]
	}
	return
}

func (z *zshComplete) SetOptions(opts map[string]string) {
	z.noDesc, _ = parseBool(opts["no-description"])
}

func (z *zshComplete) FormatCompletions(items []CompletionItem) string {
	var buf bytes.Buffer
	for _, item := range items {
		buf.WriteString(z.formatCompletion(item))
	}
	return buf.String()
}

func (z *zshComplete) formatCompletion(item CompletionItem) string {
	itemDesc := item.HelpText
	if itemDesc == "" {
		itemDesc = "_"
	}
	itemType := "plain"
	switch item.Type {
	case FileCompletionType:
		itemType = "file"
	case DirectoryCompletionType:
		itemType = "dir"
	}

	return fmt.Sprint(itemType, "\n", item.Value, "\n", itemDesc, "\n")
}

func (*zshComplete) GetSourceTemplate() *Template {
	return newSourceTemplate("zshSource", zshSourceScript)
}

func (c *CompletionContext) optionContext(opt option) *CompletionContext {
	return c.copy(c.Context.optionContext(opt))
}

func (c *CompletionContext) copy(t *Context) *CompletionContext {
	return &CompletionContext{
		Context:    t,
		Args:       c.Args,
		Incomplete: c.Incomplete,
		Bindings:   c.Bindings,
		Err:        c.Err,
	}
}

func actualCompletion(c Completion) Completion {
	if c != nil {
		return c
	}
	return CompletionFunc(nil)
}

func newSourceTemplate(name string, text string) *Template {
	t := template.New(name)
	return &Template{
		Template: template.Must(
			t.Funcs(withExecute(builtinFuncs, t)).Parse(text),
		),
		Debug: debugTemplates(),
	}
}

var (
	_ Completion = (CompletionFunc)(nil)
	_ Completion = (StandardCompletion)(0)
)
