// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
)

// Do converts action to an evaluator
func Do(a cli.Action) Evaluator {
	return evaluatorFunc(func(ctx context.Context, v any, y func(any) error) error {
		err := cli.Do(ctx, a)
		if err != nil {
			return err
		}
		return y(v)
	})
}
