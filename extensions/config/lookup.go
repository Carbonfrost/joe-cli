// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"math/big"
	"net"
	"net/url"
	"regexp"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
)

// Bool obtains the value and converts it to a bool
func (c *Config) Bool(name any) bool {
	return c.Store().Bool(name)
}

// String obtains the value and converts it to a string
func (c *Config) String(name any) string {
	return c.Store().String(name)
}

// List obtains the value and converts it to a string slice
func (c *Config) List(name any) []string {
	return c.Store().List(name)
}

// Int obtains the int for the specified name
func (c *Config) Int(name any) int {
	return c.Store().Int(name)
}

// Int8 obtains the int8 for the specified name
func (c *Config) Int8(name any) int8 {
	return c.Store().Int8(name)
}

// Int16 obtains the int16 for the specified name
func (c *Config) Int16(name any) int16 {
	return c.Store().Int16(name)
}

// Int32 obtains the int32 for the specified name
func (c *Config) Int32(name any) int32 {
	return c.Store().Int32(name)
}

// Int64 obtains the int64 for the specified name
func (c *Config) Int64(name any) int64 {
	return c.Store().Int64(name)
}

// Uint obtains the uint for the specified name
func (c *Config) Uint(name any) uint {
	return c.Store().Uint(name)
}

// Uint8 obtains the uint8 for the specified name
func (c *Config) Uint8(name any) uint8 {
	return c.Store().Uint8(name)
}

// Uint16 obtains the uint16 for the specified name
func (c *Config) Uint16(name any) uint16 {
	return c.Store().Uint16(name)
}

// Uint32 obtains the uint32 for the specified name
func (c *Config) Uint32(name any) uint32 {
	return c.Store().Uint32(name)
}

// Uint64 obtains the uint64 for the specified name
func (c *Config) Uint64(name any) uint64 {
	return c.Store().Uint64(name)
}

// Float32 obtains the float32 for the specified name
func (c *Config) Float32(name any) float32 {
	return c.Store().Float32(name)
}

// Float64 obtains the float64 for the specified name
func (c *Config) Float64(name any) float64 {
	return c.Store().Float64(name)
}

// Duration obtains the Duration for the specified name
func (c *Config) Duration(name any) time.Duration {
	return c.Store().Duration(name)
}

// File obtains the File for the specified name
func (c *Config) File(name any) *cli.File {
	return c.Store().File(name)
}

// FileSet obtains the FileSet for the specified name
func (c *Config) FileSet(name any) *cli.FileSet {
	return c.Store().FileSet(name)
}

// Map obtains the map for the specified name
func (c *Config) Map(name any) map[string]string {
	return c.Store().Map(name)
}

// NameValue obtains the value and converts it to a name-value pair
func (c *Config) NameValue(name any) *cli.NameValue {
	return c.Store().NameValue(name)
}

// NameValues obtains the value and converts it to a list of name-value pairs
func (c *Config) NameValues(name any) []*cli.NameValue {
	return c.Store().NameValues(name)
}

// URL obtains the URL for the specified name
func (c *Config) URL(name any) *url.URL {
	return c.Store().URL(name)
}

// Regexp obtains the Regexp for the specified name
func (c *Config) Regexp(name any) *regexp.Regexp {
	return c.Store().Regexp(name)
}

// IP obtains the IP for the specified name
func (c *Config) IP(name any) net.IP {
	return c.Store().IP(name)
}

// BigInt obtains the BigInt for the specified name
func (c *Config) BigInt(name any) *big.Int {
	return c.Store().BigInt(name)
}

// BigFloat obtains the BigFloat for the specified name
func (c *Config) BigFloat(name any) *big.Float {
	return c.Store().BigFloat(name)
}

// Bytes obtains the bytes for the specified name
func (c *Config) Bytes(name any) []byte {
	return c.Store().Bytes(name)
}

// Interface obtains the raw value without dereferencing
func (c *Config) Interface(name any) (any, bool) {
	return c.Store().Interface(name)
}

// Value obtains the value and converts it to Value
func (c *Config) Value(name any) any {
	return c.Store().Value(name)
}

var _ cli.Lookup = (*Config)(nil)
