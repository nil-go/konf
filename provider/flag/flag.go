// Copyright (c) 2023 The konf authors
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

	"github.com/ktong/konf/internal/maps"
)

// Flag is a Provider that loads configuration from flags.
//
// To create a new Flag, call [New].
type Flag struct {
	_         [0]func() // Ensure it's incomparable.
	set       *flag.FlagSet
	delimiter string
	prefix    string
}

// New creates a Flag with the given Option(s).
func New(opts ...Option) Flag {
	option := &options{
		delimiter: ".",
	}
	for _, opt := range opts {
		opt(option)
	}
	if option.set == nil {
		flag.Parse()
		option.set = flag.CommandLine
	}

	return Flag(*option)
}

func (f Flag) Load() (map[string]any, error) {
	values := make(map[string]any)
	f.set.VisitAll(func(flag *flag.Flag) {
		if f.prefix != "" && !strings.HasPrefix(flag.Name, f.prefix) {
			return
		}

		val := flag.Value.String()
		// Skip zero default value to avoid overriding values set by other loader.
		if val == flag.DefValue && isZeroDefValue(flag) {
			return
		}

		maps.Insert(values, strings.Split(flag.Name, f.delimiter), val)
	})

	return values, nil
}

func isZeroDefValue(flg *flag.Flag) bool {
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

	return flg.DefValue == val.Interface().(flag.Value).String() //nolint:forcetypeassert
}

func (f Flag) String() string {
	if f.prefix == "" {
		return "flag"
	}

	return "flag:" + f.prefix
}
