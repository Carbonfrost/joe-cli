// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompt

import (
	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

const confirmedDataKey = "__PromptConfirmed"

// SetConfirmed indicates the command is confirmed and that confirmation prompts
// should be skipped with the implicit answers. When used within the Users pipeline
// of a flag or arg, it provides reasonable defaults: the name --confirm
// with the optional aliases --yes and -y.
func SetConfirmed() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name:     "confirm",
			HelpText: "Assume yes to confirmation prompts",
			Value:    new(bool),
			Uses: cli.Pipeline(
				cli.OptionalAlias("yes", "y"),
				tagged,
			),
		},
		bind.Call2(setConfirmed, bind.Context(), bind.Bool()),
	)
}

func setConfirmed(c *cli.Context, confirmed bool) error {
	if !confirmed {
		return nil
	}
	c.Command().SetData(confirmedDataKey, true)
	return nil
}
