// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/hex"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/support"
)

// Map provides a config Store which uses simple key-value pairs typed
// as strings. The underlying lookup automatically provides conversions.
type Values map[string]string

func (v Values) Bool(k any) bool {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return false
	}
	p, err := support.ParseBool(value)
	if err != nil {
		return false
	}
	return p
}

func (v Values) Interface(k any) (any, bool) {
	s, ok := v[nameToString(k)]
	return s, ok
}

func (v Values) Value(k any) any {
	s, ok := v[nameToString(k)]
	if !ok {
		return nil
	}
	return s
}

func convertValue[V any, PV interface {
	cli.Value
	*V
}](v Values, k any) PV {
	value, ok := v[nameToString(k)]
	if !ok {
		return nil

	}
	result := PV(new(V))
	err := result.Set(value)
	if err != nil {
		return nil
	}
	return result
}

func (v Values) NameValue(k any) *cli.NameValue {
	return convertValue[cli.NameValue](v, k)
}

func (v Values) File(k any) *cli.File {
	return convertValue[cli.File](v, k)
}

func (v Values) FileSet(k any) *cli.FileSet {
	return convertValue[cli.FileSet](v, k)
}

func (v Values) String(k any) string {
	key := nameToString(k)
	return v[key]
}

func (v Values) List(k any) []string {
	key := nameToString(k)
	return cli.SplitList(v[key], ",", -1)
}

func (v Values) Bytes(k any) []byte {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return nil
	}
	bb, err := hex.DecodeString(value)
	if err != nil {
		return nil
	}
	return bb
}

func (v Values) Map(k any) map[string]string {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return nil
	}

	return support.FlattenValues(support.ParseMap(value))
}

func (v Values) NameValues(k any) []*cli.NameValue {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return nil
	}
	var p []*cli.NameValue
	for _, kvp := range cli.SplitList(value, ",", -1) {
		nvp := new(cli.NameValue)
		nvp.Name, nvp.Value = splitValuePair(kvp)
		p = append(p, nvp)
	}
	return p
}

func splitValuePair(arg string) (k, v string) {
	key, value, ok := support.ParseKeyValue(arg)
	if ok {
		return key, value
	}
	return key, "true"
}

func (v Values) Int(k any) int {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	i64, err := strconv.ParseInt(value, 0, strconv.IntSize)
	if err == nil {
		return int(i64)
	}
	return 0
}

func (v Values) Int8(k any) int8 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	i64, err := strconv.ParseInt(value, 0, 8)
	if err == nil {
		return int8(i64)
	}
	return 0
}

func (v Values) Int16(k any) int16 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	i64, err := strconv.ParseInt(value, 0, 16)
	if err == nil {
		return int16(i64)
	}
	return 0
}

func (v Values) Int32(k any) int32 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	i64, err := strconv.ParseInt(value, 0, 32)
	if err == nil {
		return int32(i64)
	}
	return 0
}

func (v Values) Int64(k any) int64 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	i64, err := strconv.ParseInt(value, 0, 64)
	if err == nil {
		return i64
	}
	return 0
}

func (v Values) Uint(k any) uint {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	u64, err := strconv.ParseUint(value, 0, strconv.IntSize)
	if err == nil {
		return uint(u64)
	}
	return 0
}

func (v Values) Uint8(k any) uint8 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	u64, err := strconv.ParseUint(value, 0, 8)
	if err == nil {
		return uint8(u64)
	}
	return 0
}

func (v Values) Uint16(k any) uint16 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	u64, err := strconv.ParseUint(value, 0, 16)
	if err == nil {
		return uint16(u64)
	}
	return 0
}
func (v Values) Uint32(k any) uint32 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	u64, err := strconv.ParseUint(value, 0, 32)
	if err == nil {
		return uint32(u64)
	}
	return 0
}
func (v Values) Uint64(k any) uint64 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	u64, err := strconv.ParseUint(value, 0, 64)
	if err == nil {
		return u64
	}
	return 0
}

func (v Values) Float32(k any) float32 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	f64, err := strconv.ParseFloat(value, 32)
	if err == nil {
		return float32(f64)
	}
	return 0
}

func (v Values) Float64(k any) float64 {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	f64, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return f64
	}
	return 0
}

func (v Values) Duration(k any) time.Duration {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return 0
	}
	p, _ := time.ParseDuration(value)
	return p
}

func (v Values) URL(k any) *url.URL {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return nil
	}
	p, err := url.Parse(value)
	if err == nil {
		return p
	}
	return nil
}

func (v Values) IP(k any) net.IP {
	key := nameToString(k)
	value := v[key]
	return net.ParseIP(value)
}

func (v Values) Regexp(k any) *regexp.Regexp {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return nil
	}
	p, err := regexp.Compile(value)
	if err == nil {
		return p
	}
	return nil
}

func (v Values) BigInt(k any) *big.Int {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return nil
	}
	p := new(big.Int)
	if _, ok := p.SetString(value, 10); ok {
		return p
	}
	return nil
}

func (v Values) BigFloat(k any) *big.Float {
	key := nameToString(k)
	value, ok := v[key]
	if !ok {
		return nil
	}
	p, _, err := big.ParseFloat(value, 10, 53, big.ToZero)
	if err == nil {
		return p
	}
	return nil
}

func (v Values) Has(k any) bool {
	_, ok := v[nameToString(k)]
	return ok
}

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

// Has determines whether the configuration value exists
func (c *Config) Has(name any) bool {
	return c.Store().Has(name)
}

var _ Store = (*Config)(nil)
var _ Store = (Values)(nil)
