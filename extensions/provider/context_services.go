// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

// ContextServices provides an adapter around the context to
type ContextServices struct {
	registries map[string]*Registry
}

type contextKey string

var servicesKey contextKey = "provider.services"

func init() {
	cli.Use(registerServices())
}

func registerServices() cli.Action {
	return cli.ContextValue(servicesKey, &ContextServices{
		registries: map[string]*Registry{},
	})
}

// Services gets the context services for working with providers.
// This function panics if the context does not contain context services,
// which are initialized with the app
func Services(c context.Context) *ContextServices {
	return c.Value(servicesKey).(*ContextServices)
}

// Registry gets the registry by name, if any. The name argument
// is the name of the registry, but as a special case, if the name
// starts with dashes as if the name of a flag, those are trimmed.
// The argument can also be a Flag. The name of the flag is used,
// or registry specified by the flag's Value.
func (c *ContextServices) LookupRegistry(name any) (*Registry, bool) {
	v := registryName(name)
	r, ok := c.registries[v]
	return r, ok
}

// New retrieves the given provider and invokes it
func (c *ContextServices) New(registry any, provider string, args map[string]string) (any, error) {
	r, ok := c.LookupRegistry(registry)
	if !ok {
		return nil, fmt.Errorf("registry not found: %q", registryName(registry))
	}
	return r.New(provider, args)
}

func registryName(name any) string {
	switch v := name.(type) {
	case string:
		return strings.TrimPrefix(v, "-")

	case *cli.Flag:
		if value, ok := v.Value.(*Value); ok {
			return cmp.Or(value.Registry, v.Name)
		}
		return v.Name
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}
}
