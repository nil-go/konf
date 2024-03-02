// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package pflag loads configuration from flags defined by [spf13/pflag].
//
// PFlag loads flags in [pflag.CommandLine] whose names starts with the given prefix
// and returns them as a nested map[string]any.
// The unchanged flags with zero default value are skipped to avoid
// overriding values set by other loader.
//
// It splits the names by delimiter. For example, with the default delimiter ".",
// the flag `parent.child.key="1"` is loaded as `{parent: {child: {key: "1"}}}`.
package pflag

import (
	"flag"
	"reflect"
	"strings"

	"github.com/spf13/pflag"

	"github.com/nil-go/konf/provider/pflag/internal/maps"
)

// PFlag is a Provider that loads configuration from flags defined by [spf13/pflag].
//
// To create a new PFlag, call [New].
type PFlag struct {
	konf     konf
	prefix   string
	set      *pflag.FlagSet
	splitter func(string) []string
}

// New creates a PFlag with the given Option(s).
//
// The first parameter is the konf Config instance that checks if the defined flags
// have been set by other providers. If not, default flag values are merged.
// If they exist, flag values are merged only if explicitly set in the command line.
func New(konf konf, opts ...Option) PFlag {
	option := &options{
		konf: konf,
	}
	for _, opt := range opts {
		opt(option)
	}

	return PFlag(*option)
}

func (f PFlag) Load() (map[string]any, error) { //nolint:cyclop
	set := f.set
	if set == nil {
		if !pflag.Parsed() {
			pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
			pflag.Parse()
		}
		set = pflag.CommandLine
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
	set.VisitAll(
		func(flag *pflag.Flag) {
			if f.prefix != "" && !strings.HasPrefix(flag.Name, f.prefix) {
				return
			}

			keys := splitter(flag.Name)
			if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
				return
			}

			// Skip zero default value to avoid overriding values set by other loader.
			if !flag.Changed && (exists(keys) || zeroDefaultValue(flag)) {
				return
			}

			maps.Insert(values, keys, flag.Value.String())
		},
	)

	return values, nil
}

func zeroDefaultValue(flag *pflag.Flag) bool { //nolint:cyclop
	switch flag.Value.Type() {
	case "bool":
		return flag.DefValue == "false"
	case "duration":
		// Beginning in Go 1.7, duration zero values are "0s"
		return flag.DefValue == "0" || flag.DefValue == "0s"
	case "count", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return flag.DefValue == "0"
	case "string":
		return flag.DefValue == ""
	case "ip", "ipMask", "ipNet":
		return flag.DefValue == "<nil>"
	case
		"boolSlice", "durationSlice", "ipSlice",
		"bytesHex", "bytesBase64", "stringArray", "stringSlice", "stringToString", "stringToInt", "stringToInt64",
		"intSlice", "int32Slice", "int64Slice", "uintSlice", "float32Slice", "float64Slice":
		return flag.DefValue == "[]"
	default:
		switch flag.DefValue {
		case "false", "<nil>", "", "0":
			return true
		default:
			return false
		}
	}
}

func (f PFlag) String() string {
	if f.prefix == "" {
		return "pflag"
	}

	return "pflag:" + f.prefix
}

type konf interface {
	Exists([]string) bool
}
