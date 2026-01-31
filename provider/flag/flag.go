// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package flag loads configuration from flags defined by [flag].
//
// Flag loads flags in [flag.CommandLine] whose names starts with the given prefix
// and returns them as a nested map[string]any.
// The unchanged flags with zero default value are skipped to avoid
// overriding values set by other loader.
//
// It splits the names by delimiter. For example, with the default delimiter ".",
// the flag `parent.child.key="1"` is loaded as `{parent: {child: {key: "1"}}}`.
package flag

import (
	"flag"
	"reflect"
	"strings"

	"github.com/nil-go/konf/internal/maps"
)

// Flag is a Provider that loads configuration from flags.
//
// To create a new Flag, call [New].
type Flag struct {
	konf     konf
	prefix   string
	set      *flag.FlagSet
	splitter func(string) []string
}

// New creates a Flag with the given Option(s).
//
// The first parameter is the konf Config instance that checks if the defined flags
// have been set by other providers. If not, default flag values are merged.
// If they exist, flag values are merged only if explicitly set in the command line.
func New(konf konf, opts ...Option) Flag {
	option := &options{
		konf: konf,
	}
	for _, opt := range opts {
		opt(option)
	}

	return Flag(*option)
}

func (f Flag) Load() (map[string]any, error) { //nolint:cyclop
	set := f.set
	if set == nil {
		if !flag.Parsed() {
			flag.Parse()
		}
		set = flag.CommandLine
	}

	splitter := f.splitter
	if splitter == nil {
		splitter = func(s string) []string {
			return strings.Split(s, ".")
		}
	}

	var exists func([]string) bool
	if f.konf != nil && !reflect.ValueOf(f.konf).IsNil() {
		exists = f.konf.Exists
	} else {
		exists = func([]string) bool {
			return false
		}
	}

	values := make(map[string]any)
	set.VisitAll(func(flg *flag.Flag) {
		if f.prefix != "" && !strings.HasPrefix(flg.Name, f.prefix) {
			return
		}

		keys := splitter(flg.Name)
		if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
			return
		}

		value := flg.Value.String()
		// Skip zero default value to avoid overriding values set by other loader.
		if value == flg.DefValue && (exists(keys) || isZeroValue(flg)) {
			return
		}

		if getter, ok := flg.Value.(flag.Getter); ok {
			maps.Insert(values, keys, getter.Get())
		} else {
			maps.Insert(values, keys, value)
		}
	})

	return values, nil
}

// isZeroValue is copied from flag/flag.go.
func isZeroValue(flg *flag.Flag) bool {
	// Build a zero value of the flag's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	typ := reflect.TypeOf(flg.Value)
	var val reflect.Value
	if typ.Kind() == reflect.Pointer {
		val = reflect.New(typ.Elem())
	} else {
		val = reflect.Zero(typ)
	}

	return flg.DefValue == val.Interface().(flag.Value).String() //nolint:errcheck,forcetypeassert
}

func (f Flag) String() string {
	return "flag:" + f.prefix + "*"
}

type konf interface {
	Exists([]string) bool
}
