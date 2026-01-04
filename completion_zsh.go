// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

type zshComplete struct {
	noDesc bool
}

const zshSourceScript = `

#compdef {{ .App.Name }}
{{ .CompletionFunc }}() {
    local -a completions
    local -a completions_with_descriptions
    local -a response

    (( ! $+commands[{{ .App.Name }}] )) && return 1

    response=("${(@f)$(env COMP_WORDS="${words[*]}" COMP_CWORD=$((CURRENT-1)) \
{{ .JoeCompletionVar }}=zsh {{ .App.Name }})}")
    for type key descr no_space in ${response}; do
        if [[ -n "$no_space" ]]; then
            space="-S ''"
        fi
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
        eval _describe -V unsorted completions_with_descriptions -U $space
    fi
    if [ -n "$completions" ]; then
        compadd -U -V unsorted -a completions $space
    fi
}
compdef {{ .CompletionFunc }} {{ .App.Name }};
`

func newZshComplete() ShellComplete {
	return &zshComplete{}
}

func (*zshComplete) GetCompletionRequest() (args []string, incomplete string) {
	cwords, _ := Split(os.Getenv("COMP_WORDS"))
	cword, _ := strconv.Atoi(os.Getenv("COMP_CWORD"))

	if cword <= len(cwords) {
		args = cwords[0:cword]
	}
	if cword < len(cwords) {
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
	case CompletionTypeFile:
		itemType = "file"
	case CompletionTypeDirectory:
		itemType = "dir"
	}
	spaceAfter := "1"
	if item.PreventSpaceAfter {
		spaceAfter = ""
	}

	return fmt.Sprint(itemType, "\n", item.Value, "\n", itemDesc, "\n", spaceAfter, "\n")
}

func (*zshComplete) GetSourceTemplate() *Template {
	return newSourceTemplate("zshSource", zshSourceScript)
}
