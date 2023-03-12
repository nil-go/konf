// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package flag loads configuration from flags.
package flag

import (
	"flag"
	"reflect"
	"strings"

	"github.com/ktong/konf/internal/maps"
)

// Flag is a Provider that loads configuration from flags.
//
// The name of flags is case-insensitive.
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
	config := make(map[string]any)
	f.set.VisitAll(func(flag *flag.Flag) {
		if f.prefix != "" && !strings.HasPrefix(flag.Name, f.prefix) {
			return
		}

		// Skip zero default value to avoid overriding values set by other loader.
		if flag.Value.String() == flag.DefValue && isZeroDefValue(flag) {
			return
		}

		maps.Insert(config, strings.Split(strings.ToLower(flag.Name), f.delimiter), flag.Value.String())
	})

	return config, nil
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
