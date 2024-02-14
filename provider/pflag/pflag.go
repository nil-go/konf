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

type konf interface {
	Exists([]string) bool
}

// New creates a PFlag with the given Option(s).
//
// The first parameter is the konf Config instance that checks if the defined flags
// have been set by other providers. If not, default flag values are merged.
// If they exist, flag values are merged only if explicitly set in the command line.
//
// It panics if the konf is nil.
func New(konf konf, opts ...Option) PFlag {
	if konf == nil {
		panic("cannot create Flag with nil konf")
	}

	option := &options{
		konf: konf,
	}
	for _, opt := range opts {
		opt(option)
	}
	if option.splitter == nil {
		option.splitter = func(s string) []string { return strings.Split(s, ".") }
	}
	if option.set == nil {
		if !pflag.Parsed() {
			pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
			pflag.Parse()
		}
		option.set = pflag.CommandLine
	}

	return PFlag(*option)
}

func (f PFlag) Load() (map[string]any, error) {
	values := make(map[string]any)
	f.set.VisitAll(
		func(flag *pflag.Flag) {
			if f.prefix != "" && !strings.HasPrefix(flag.Name, f.prefix) {
				return
			}

			keys := f.splitter(flag.Name)
			if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
				return
			}

			val := f.flagVal(flag)
			// Skip zero default value to avoid overriding values set by other loader.
			if !flag.Changed && (f.konf.Exists(keys) || reflect.ValueOf(val).IsZero()) {
				return
			}

			maps.Insert(values, keys, val)
		},
	)

	return values, nil
}

//nolint:cyclop,funlen,gocyclo,nlreturn
func (f PFlag) flagVal(flag *pflag.Flag) any {
	switch flag.Value.Type() {
	case "int":
		i, _ := f.set.GetInt(flag.Name)
		return int64(i)
	case "uint":
		i, _ := f.set.GetUint(flag.Name)
		return uint64(i)
	case "int8":
		i, _ := f.set.GetInt8(flag.Name)
		return int64(i)
	case "uint8":
		i, _ := f.set.GetUint8(flag.Name)
		return uint64(i)
	case "int16":
		i, _ := f.set.GetInt16(flag.Name)
		return int64(i)
	case "uint16":
		i, _ := f.set.GetUint16(flag.Name)
		return uint64(i)
	case "int32":
		i, _ := f.set.GetInt32(flag.Name)
		return int64(i)
	case "uint32":
		i, _ := f.set.GetUint32(flag.Name)
		return uint64(i)
	case "int64":
		val, _ := f.set.GetInt64(flag.Name)
		return val
	case "uint64":
		val, _ := f.set.GetUint64(flag.Name)
		return val
	case "float":
		val, _ := f.set.GetFloat64(flag.Name)
		return val
	case "float32":
		val, _ := f.set.GetFloat32(flag.Name)
		return val
	case "float64":
		val, _ := f.set.GetFloat64(flag.Name)
		return val
	case "bool":
		val, _ := f.set.GetBool(flag.Name)
		return val
	case "duration":
		val, _ := f.set.GetDuration(flag.Name)
		return val
	case "ip":
		val, _ := f.set.GetIP(flag.Name)
		return val
	case "ipMask":
		val, _ := f.set.GetIPv4Mask(flag.Name)
		return val
	case "ipNet":
		val, _ := f.set.GetIPNet(flag.Name)
		return val
	case "count":
		val, _ := f.set.GetCount(flag.Name)
		return val
	case "bytesHex":
		val, _ := f.set.GetBytesHex(flag.Name)
		return val
	case "bytesBase64":
		val, _ := f.set.GetBytesBase64(flag.Name)
		return val
	case "string":
		val, _ := f.set.GetString(flag.Name)
		return val
	case "stringSlice":
		val, _ := f.set.GetStringSlice(flag.Name)
		return val
	case "intSlice":
		val, _ := f.set.GetIntSlice(flag.Name)
		return val
	case "uintSlice":
		val, _ := f.set.GetUintSlice(flag.Name)
		return val
	case "int32Slice":
		val, _ := f.set.GetInt32Slice(flag.Name)
		return val
	case "int64Slice":
		val, _ := f.set.GetInt64Slice(flag.Name)
		return val
	case "float32Slice":
		val, _ := f.set.GetFloat32Slice(flag.Name)
		return val
	case "float64Slice":
		val, _ := f.set.GetFloat64Slice(flag.Name)
		return val
	case "boolSlice":
		val, _ := f.set.GetBoolSlice(flag.Name)
		return val
	case "durationSlice":
		val, _ := f.set.GetDurationSlice(flag.Name)
		return val
	case "ipSlice":
		val, _ := f.set.GetIPSlice(flag.Name)
		return val
	case "stringArray":
		val, _ := f.set.GetStringArray(flag.Name)
		return val
	case "stringToString":
		val, _ := f.set.GetStringToString(flag.Name)
		return val
	case "stringToInt":
		val, _ := f.set.GetStringToInt(flag.Name)
		return val
	case "stringToInt64":
		val, _ := f.set.GetStringToInt64(flag.Name)
		return val
	default:
		return flag.Value.String()
	}
}

func (f PFlag) String() string {
	if f.prefix == "" {
		return "pflag"
	}

	return "pflag:" + f.prefix
}
