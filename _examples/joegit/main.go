// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"os"

	"github.com/Carbonfrost/joe-cli"
)

func main() {
	app := &cli.App{
		Name: "joegit",
		Uses: categorizeCommands,
		Flags: []*cli.Flag{
			{Name: "C", UsageText: "path"},
			{Name: "c", UsageText: "name=value"},
			{Name: "exec-path", UsageText: "path"},
			{Name: "html-path", Value: new(bool)},
			{Name: "man-path", Value: new(bool)},
			{Name: "paginate", Aliases: []string{"p"}, Value: new(bool)},
			{Name: "no-pager", Aliases: []string{"P"}, Value: new(bool)},
			{Name: "no-replace-objects", Value: new(bool)},
			{Name: "bare", Value: new(bool)},
			{Name: "git-path", UsageText: "path"},
			{Name: "work-tree-path", UsageText: "path"},
			{Name: "namespace", UsageText: "name"},
			{Name: "super-prefix", UsageText: "path"},
			{Name: "config-env", UsageText: "name=envvar"},
		},
		Commands: []*cli.Command{
			{Name: "clone", HelpText: "Clone a repository into a new directory"},
			{Name: "init", HelpText: "Create an empty Git repository or reinitialize an existing one"},
			{Name: "add", HelpText: "Add file contents to the index"},
			{Name: "mv", HelpText: "Move or rename a file, a directory, or a symlink"},
			{Name: "restore", HelpText: "Restore working tree files"},
			{Name: "rm", HelpText: "Remove files from the working tree and from the index"},
			{Name: "sparse-checkout", HelpText: "Initialize and modify the sparse-checkout"},
			{Name: "bisect", HelpText: "Use binary search to find the commit that introduced a bug"},
			{Name: "diff", HelpText: "Show changes between commits, commit and working tree, etc"},
			{Name: "grep", HelpText: "Print lines matching a pattern"},
			{Name: "log", HelpText: "Show commit logs"},
			{Name: "show", HelpText: "Show various types of objects"},
			{Name: "status", HelpText: "Show the working tree status"},
			{Name: "branch", HelpText: "List, create, or delete branches"},
			{Name: "commit", HelpText: "Record changes to the repository"},
			{Name: "merge", HelpText: "Join two or more development histories together"},
			{Name: "rebase", HelpText: "Reapply commits on top of another base tip"},
			{Name: "reset", HelpText: "Reset current HEAD to the specified state"},
			{Name: "switch", HelpText: "Switch branches"},
			{Name: "tag", HelpText: "Create, list, delete or verify a tag object signed with GPG"},
			{Name: "fetch", HelpText: "Download objects and refs from another repository"},
			{Name: "pull", HelpText: "Fetch from and integrate with another repository or a local branch"},
			{Name: "push", HelpText: "Update remote refs along with associated objects"},
		},
		Description: `These are common Git commands used in various situations.
'git help -a' and 'git help -g' list available subcommands and some
concept guides. See 'git help <command>' or 'git help <concept>'
to read about a specific subcommand or concept.
See 'git help git' for an overview of the system.`,
	}

	app.Run(os.Args)
}

const (
	startCategory       = "start a working area"
	workCategory        = "work on the current change"
	historyCategory     = "examine the history and state"
	growCategory        = "grow, mark and tweak your common history"
	collaborateCategory = "collaborate"
)

var (
	categoryMap = map[string]string{
		"clone": startCategory,
		"init":  startCategory,

		"add":             workCategory,
		"mv":              workCategory,
		"restore":         workCategory,
		"rm":              workCategory,
		"sparse-checkout": workCategory,

		"bisect": historyCategory,
		"diff":   historyCategory,
		"grep":   historyCategory,
		"log":    historyCategory,
		"show":   historyCategory,
		"status": historyCategory,

		"branch": growCategory,
		"commit": growCategory,
		"merge":  growCategory,
		"rebase": growCategory,
		"reset":  growCategory,
		"switch": growCategory,
		"tag":    growCategory,

		"fetch": collaborateCategory,
		"pull":  collaborateCategory,
		"push":  collaborateCategory,
	}
)

func categorizeCommands(c *cli.Context) error {
	for _, cmd := range c.Command().Subcommands {
		cmd.Category = categoryMap[cmd.Name]
	}

	return nil
}
