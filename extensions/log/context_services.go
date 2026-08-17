// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	cli "github.com/Carbonfrost/joe-cli"
)

// ContextServices provides an adapter around the context which tracks the
// loggers that have been registered with the app.  Each app gets its own
// instance, which is initialized with the app.
type ContextServices struct {
	mu      sync.RWMutex
	loggers map[string]*Logger
}

// Context keys for the context service or for each of the loggers
type (
	contextKey string
	loggerKey  string
)

var servicesKey contextKey = "log.services"

// binding associates the context services with the app they belong to so that
// the log functions which don't take a context can bridge to the current app.
type binding struct {
	app      *cli.App
	services *ContextServices
}

var currentBinding atomic.Pointer[binding]

func init() {
	cli.Extend(registerServices())
}

func registerServices() cli.Action {
	return cli.WithContext(func(ctx context.Context) context.Context {
		// This thunk ensures that a new instance is created per app
		services := &ContextServices{
			loggers: map[string]*Logger{},
		}
		if app := cli.CurrentApp(); app != nil {
			currentBinding.Store(&binding{app: app, services: services})
		}
		return context.WithValue(ctx, servicesKey, services)
	})
}

// Services gets the context services for working with loggers.
// This function panics if the context does not contain context services,
// which are initialized with the app
func Services(c context.Context) *ContextServices {
	return c.Value(servicesKey).(*ContextServices)
}

// Register adds the logger to the services using its name.  Any logger which
// was previously registered with the same name is replaced.
func (c *ContextServices) Register(l *Logger) {
	name := l.Name()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.loggers[name] = l
}

// Lookup obtains the logger which was registered with the given name, if any.
// The default logger is registered with the empty string as its name.
func (c *ContextServices) Lookup(name string) (*Logger, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	l, ok := c.loggers[name]
	return l, ok
}

func register(ctx context.Context, l *Logger) {
	s, ok := tryServices(ctx)
	if !ok {
		s, ok = currentServices()
	}
	if ok {
		s.Register(l)
	}
}

func tryServices(c context.Context) (*ContextServices, bool) {
	if c == nil {
		return nil, false
	}
	s, ok := c.Value(servicesKey).(*ContextServices)
	return s, ok
}

func currentServices() (*ContextServices, bool) {
	app := cli.CurrentApp()
	if app == nil {
		return nil, false
	}
	b := currentBinding.Load()
	if b == nil || b.app != app {
		return nil, false
	}
	return b.services, true
}

func currentLogger() *slog.Logger {
	if s, ok := currentServices(); ok {
		if l, ok := s.Lookup(""); ok {
			return l.Logger()
		}
	}
	return slog.Default()
}

func contextLogger(ctx context.Context) *slog.Logger {
	if l, err := tryFromContext(ctx, ""); err == nil {
		return l.Logger()
	}
	return currentLogger()
}
