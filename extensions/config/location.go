// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"strconv"
)

// Location defines a representation of configuration file locations
type Location interface {
	// Paths resolves the file paths for this location
	Paths(context.Context) ([]string, error)
}

// IdiomaticLocation defines a location which exists in a particular layer
type IdiomaticLocation interface {
	Location

	// Layer gets the layer where the location is found
	Layer() Layer

	// OS gets the operating system the location supports
	OS() string

	// Arch gets the architecture the location supports
	Arch() string
}

// IdiomaticLocationProvider provides an interface for how idiomatic locations
// are detected
type IdiomaticLocationProvider interface {
	Location

	// Resolve determines the locations
	Resolve(Options) ([]IdiomaticLocation, error)

	// FindProfileNames locates available profile names
	FindProfileNames(context.Context) []string
}

// Layer identifies layers in the configuration system
type Layer int

// Layers where a location exists
const (
	LayerUnspecified Layer = -1
	LayerIntrinsic   Layer = 0
	LayerSystem      Layer = 2
	LayerUser        Layer = 4
	LayerWorkspace   Layer = 6
	LayerProfile     Layer = 8
	LayerAdditional  Layer = 10
)

var (
	layerLabels = map[Layer]string{
		LayerUnspecified: "UNSPECIFIED",
		LayerIntrinsic:   "INTRINSIC",
		LayerSystem:      "SYSTEM",
		LayerUser:        "USER",
		LayerWorkspace:   "WORKSPACE",
		LayerProfile:     "PROFILE",
		LayerAdditional:  "ADDITIONAL",
	}
)

func (l Layer) String() string {
	l = max(LayerUnspecified, min(LayerAdditional, l))
	if res, ok := layerLabels[l]; ok {
		return res
	}
	if res, ok := layerLabels[l-1]; ok {
		return res + "+1"
	}
	return strconv.Itoa(int(l))
}
