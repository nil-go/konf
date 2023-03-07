// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

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
// It skips it if the flag's value is same as default zero value.
type Flag struct {
	set       *flag.FlagSet
	delimiter string
	prefix    string
}

// New returns a Flag with the given Option(s).
func New(opts ...Option) Flag {
	flg := Flag{
		delimiter: ".",
	}
	for _, opt := range opts {
		opt(&flg)
	}

	if flg.set == nil {
		flag.Parse()
		flg.set = flag.CommandLine
	}

	return flg
}

func (f Flag) Load() (map[string]any, error) {
	config := make(map[string]any)
	f.set.VisitAll(func(flag *flag.Flag) {
		if f.prefix != "" && !strings.HasPrefix(flag.Name, f.prefix) {
			return
		}

		// Skip zero default value to avoid overriding values set by other loader.
		if flag.Value.String() == flag.DefValue && isZeroValue(flag) {
			return
		}

		maps.Insert(config, strings.Split(strings.ToLower(flag.Name), f.delimiter), flag.Value.String())
	})

	return config, nil
}

// isZeroValue determines whether the flag has the zero value.
func isZeroValue(flg *flag.Flag) bool {
	// Build a zero value of the flag's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	typ := reflect.TypeOf(flg.Value)
	var zero reflect.Value
	if typ.Kind() == reflect.Pointer {
		zero = reflect.New(typ.Elem())
	} else {
		zero = reflect.Zero(typ)
	}

	// Catch panics calling the String method.
	defer func() {
		_ = recover()
	}()

	return flg.DefValue == zero.Interface().(flag.Value).String() //nolint:forcetypeassert
}
