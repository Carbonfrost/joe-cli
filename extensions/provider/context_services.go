// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package provider

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
)

// ContextServices provides an adapter around the context to
type ContextServices struct {
	registries map[string]*Registry
}

type contextKey string

var servicesKey contextKey = "provider.services"

// WithServices obtains the context services from the specified context.
// If they do not exist, they are added and the context result is returned.
func WithServices(c context.Context) (context.Context, *ContextServices) {
	o := c.Value(servicesKey)
	if o == nil {
		res := &ContextServices{
			registries: map[string]*Registry{},
		}
		return context.WithValue(c, servicesKey, res), res
	}
	return c, o.(*ContextServices)
}

// Services obtains the contextual services used by the package.  If not
// already present, it will be added to the context.
func Services(c *cli.Context) *ContextServices {
	newCtx, o := WithServices(c.Context())
	c.SetContext(newCtx)
	return o
}

// Registry gets the registry by name, if any
func (c *ContextServices) Registry(name string) *Registry {
	return c.registries[name]
}
