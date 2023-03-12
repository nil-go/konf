// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package flag loads configuration from flags.
//
// Flag loads all flags in [flag.CommandLine] and returns nested map[string]any.
// by splitting the names by `.`.E.g. the flag `parent.child.key` with value 1
// is loaded as `{parent: {child: {key: 1}}}`.
// The unchanged flags with zero default value are skipped to avoid
// overriding values set by other loader.
//
// The default behavior can be changed with following options:
//   - WithPrefix enables loads flags with the given prefix in the name.
//   - WithFlagSet provides the flag set that loads configuration from.
//   - WithDelimiter provides the delimiter when splitting flag name to nested keys.
package flag

import (
	"flag"
	"reflect"
	"strings"

	"github.com/ktong/konf/internal/maps"
)

// Flag is a Provider that loads configuration from flags.
type Flag struct {
	_         [0]func() // Ensure it's incomparable.
	set       *flag.FlagSet
	delimiter string
	prefix    string
}

// New returns a Flag with the given Option(s).
func New(opts ...Option) Flag {
	option := apply(opts)
	if option.set == nil {
		flag.Parse()
		option.set = flag.CommandLine
	}

	return Flag(option)
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
