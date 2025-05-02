// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package expr // intentional

func IsVisible(e *Expr) bool {
	return !e.internalFlags().hidden()
}
