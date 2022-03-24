package provider

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
)

// ContextServices provides an adapter around the context to
type ContextServices struct {
	*cli.Context
	registries map[string]*Registry
}

type contextKey string

var servicesKey contextKey = "provider.services"

// Services obtains the contextual services used by the package.  If not
// already present, it will be added to the context.
func Services(c *cli.Context) *ContextServices {
	o := c.Context.Value(servicesKey)
	if o == nil {
		res := &ContextServices{
			Context:    c,
			registries: map[string]*Registry{},
		}
		c.Context = context.WithValue(c.Context, servicesKey, res)
		return res
	}
	return o.(*ContextServices)
}

// Registry gets the registry by name, if any
func (c *ContextServices) Registry(name string) *Registry {
	return c.registries[name]
}
