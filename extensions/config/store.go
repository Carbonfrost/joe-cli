// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"

	cli "github.com/Carbonfrost/joe-cli"
)

// Store provides configuration storage based on the Lookup interface.
// It retrieves values by qualified names delimited by periods, using
// a "dig" algorithm to traverse hierarchical names.
type Store interface {
	cli.Lookup
}

// Loader loads the configuration system
type Loader interface {
	Load(context.Context) (Store, error)
}
