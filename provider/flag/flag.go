// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package flag

import (
	"errors"
	"flag"
	"fmt"
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
	option := apply(opts)
	if option.set == nil {
		flag.Parse()
		option.set = flag.CommandLine
	}

	return Flag(option)
}

func (f Flag) Load() (map[string]any, error) {
	var errs []error
	config := make(map[string]any)
	f.set.VisitAll(func(flag *flag.Flag) {
		if f.prefix != "" && !strings.HasPrefix(flag.Name, f.prefix) {
			return
		}

		// Skip zero default value to avoid overriding values set by other loader.
		if flag.Value.String() == flag.DefValue {
			zero, err := isZeroValue(flag, flag.DefValue)
			if err != nil {
				errs = append(errs, err)
			}
			if zero {
				return
			}
		}

		maps.Insert(config, strings.Split(strings.ToLower(flag.Name), f.delimiter), flag.Value.String())
	})

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return config, nil
}

// isZeroValue determines whether the string represents the zero
// value for a flag.
func isZeroValue(flg *flag.Flag, value string) (ok bool, err error) { //nolint:nonamedreturns
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

	// Catch panics calling the String method, which shouldn't prevent the
	// usage message from being printed, but that we should report to the
	// user so that they know to fix their code.
	defer func() {
		if msg := recover(); msg != nil {
			if typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}
			err = fmt.Errorf( //nolint:goerr113
				"panic calling String method on zero %v for flag %s: %v",
				typ, flg.Name, msg,
			)
		}
	}()

	return value == val.Interface().(flag.Value).String(), nil //nolint:forcetypeassert
}
