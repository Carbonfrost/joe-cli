// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

import (
	"context"
	"encoding/json"
	"os"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
)

func Dump(app *cli.App) {
	app.Initialize(context.Background())

	m := marshal.From(app)
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "    ")
	e.Encode(m)
}
