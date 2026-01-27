// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config // intentional

import "context"

func CompleteSetup(ctx context.Context, ws *Workspace) {
	ws.completeSetup(ctx)
}
