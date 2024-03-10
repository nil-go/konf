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

			val, _ := flagVal(set, flag) // Ignore error as it uses whatever returned.
			maps.Insert(values, keys, val)
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
		"float", "float32", "float64":
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

//nolint:cyclop,funlen,gocyclo,wrapcheck
func flagVal(set *pflag.FlagSet, flag *pflag.Flag) (any, error) {
	switch flag.Value.Type() {
	case "bool":
		return set.GetBool(flag.Name)
	case "duration":
		return set.GetDuration(flag.Name)
	case "count":
		return set.GetCount(flag.Name)
	case "int":
		return set.GetInt(flag.Name)
	case "int8":
		return set.GetInt8(flag.Name)
	case "int16":
		return set.GetInt16(flag.Name)
	case "int32":
		return set.GetInt32(flag.Name)
	case "int64":
		return set.GetInt64(flag.Name)
	case "uint":
		return set.GetUint(flag.Name)
	case "uint8":
		return set.GetUint8(flag.Name)
	case "uint16":
		return set.GetUint16(flag.Name)
	case "uint32":
		return set.GetUint32(flag.Name)
	case "uint64":
		return set.GetUint64(flag.Name)
	case "float":
		return set.GetFloat64(flag.Name)
	case "float32":
		return set.GetFloat32(flag.Name)
	case "float64":
		return set.GetFloat64(flag.Name)
	case "string":
		return set.GetString(flag.Name)
	case "ip":
		return set.GetIP(flag.Name)
	case "ipMask":
		return set.GetIPv4Mask(flag.Name)
	case "ipNet":
		return set.GetIPNet(flag.Name)
	case "boolSlice":
		return set.GetBoolSlice(flag.Name)
	case "durationSlice":
		return set.GetDurationSlice(flag.Name)
	case "ipSlice":
		return set.GetIPSlice(flag.Name)
	case "bytesHex":
		return set.GetBytesHex(flag.Name)
	case "bytesBase64":
		return set.GetBytesBase64(flag.Name)
	case "stringArray":
		return set.GetStringArray(flag.Name)
	case "stringSlice":
		return set.GetStringSlice(flag.Name)
	case "stringToString":
		return set.GetStringToString(flag.Name)
	case "stringToInt":
		return set.GetStringToInt(flag.Name)
	case "stringToInt64":
		return set.GetStringToInt64(flag.Name)
	case "intSlice":
		return set.GetIntSlice(flag.Name)
	case "int32Slice":
		return set.GetInt32Slice(flag.Name)
	case "int64Slice":
		return set.GetInt64Slice(flag.Name)
	case "uintSlice":
		return set.GetUintSlice(flag.Name)
	case "float32Slice":
		return set.GetFloat32Slice(flag.Name)
	case "float64Slice":
		return set.GetFloat64Slice(flag.Name)
	default:
		return flag.Value.String(), nil
	}
}

func (f PFlag) String() string {
	return "pflag:" + f.prefix + "*"
}

type konf interface {
	Exists([]string) bool
}
