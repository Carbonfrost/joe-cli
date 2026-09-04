// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value

import (
	"fmt"
	"strings"
	"time"
)

// Time provides a value whose parsing can be configured, which increases
// the ergonomics of dates and times in flags and args.
type Time struct {
	value time.Time

	// AllowedLayouts specifies the layouts that are allowed when parsing
	// the timestamps. By default, DefaultTimeLayouts is used when this
	// is unspecified.
	AllowedLayouts []string

	dateOnly bool
}

// DateOnly indicates whether the timestamp was parsed only from a date, which
// means that the time of day that was intended by the user could be ambiguous.
func (t Time) DateOnly() bool {
	return t.dateOnly
}

// Reset will reset the timestamp. The configuration attribute, AllowedLayout
// is not modified by this method
func (t *Time) Reset() {
	t.value = time.Time{}
	t.dateOnly = false
}

// DefaultTimeLayouts specifies the layouts that are allowed when parsing
// Time. The default specifies RFC3339, which is what the built-in parser would
// use, plus bare dates (YYYY-MM-dd) and their time of day using the 24-hour clock
// and optionally with seconds resolution.
var DefaultTimeLayouts = []string{
	time.DateOnly,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
}

func (t *Time) Set(arg string) error {
	for _, layout := range t.layouts() {
		if value, err := time.ParseInLocation(layout, arg, time.Local); err == nil {
			t.value, t.dateOnly = value, !hasTimeComponents(layout)
			return nil
		}
	}
	return fmt.Errorf("not a valid date or time: %q", arg)
}

func (t *Time) layouts() []string {
	if len(t.AllowedLayouts) == 0 {
		return DefaultTimeLayouts
	}
	return t.AllowedLayouts
}

func hasTimeComponents(layout string) bool {
	var timeComponents = []string{
		"15", "3", "03", "4", "04", "5", "05", "PM",
	}
	for _, t := range timeComponents {
		if strings.Contains(layout, t) {
			return true
		}
	}
	return false
}

func (t *Time) String() string {
	switch {
	case t.value.IsZero():
		return ""
	case t.dateOnly:
		return t.value.Format(time.DateOnly)
	}
	return t.value.Format(time.RFC3339)
}

// Value obtains the timestamp as time.Time.
func (t *Time) Value() time.Time {
	return t.value
}

// Range interprets the timestamp as a range of time. If an exact timestamp was
// specified or if any component of the time of day was specified, both return
// values will correspond to the value that was parsed. If only a date was
// specified the range will correspond to the start and end of day.
func (t *Time) Range() (start, end time.Time) {
	if t.dateOnly {
		return t.value, t.value.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return t.value, t.value
}
