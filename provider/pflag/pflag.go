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

			val := f.flagVal(set, flag)
			// Skip zero default value to avoid overriding values set by other loader.
			if !flag.Changed && (exists(keys) || reflect.ValueOf(val).IsZero()) {
				return
			}

			maps.Insert(values, keys, val)
		},
	)

	return values, nil
}

//nolint:cyclop,funlen,gocyclo,nlreturn
func (f PFlag) flagVal(set *pflag.FlagSet, flag *pflag.Flag) any {
	switch flag.Value.Type() {
	case "int":
		i, _ := set.GetInt(flag.Name)
		return int64(i)
	case "uint":
		i, _ := set.GetUint(flag.Name)
		return uint64(i)
	case "int8":
		i, _ := set.GetInt8(flag.Name)
		return int64(i)
	case "uint8":
		i, _ := set.GetUint8(flag.Name)
		return uint64(i)
	case "int16":
		i, _ := set.GetInt16(flag.Name)
		return int64(i)
	case "uint16":
		i, _ := set.GetUint16(flag.Name)
		return uint64(i)
	case "int32":
		i, _ := set.GetInt32(flag.Name)
		return int64(i)
	case "uint32":
		i, _ := set.GetUint32(flag.Name)
		return uint64(i)
	case "int64":
		val, _ := set.GetInt64(flag.Name)
		return val
	case "uint64":
		val, _ := set.GetUint64(flag.Name)
		return val
	case "float":
		val, _ := set.GetFloat64(flag.Name)
		return val
	case "float32":
		val, _ := set.GetFloat32(flag.Name)
		return val
	case "float64":
		val, _ := set.GetFloat64(flag.Name)
		return val
	case "bool":
		val, _ := set.GetBool(flag.Name)
		return val
	case "duration":
		val, _ := set.GetDuration(flag.Name)
		return val
	case "ip":
		val, _ := set.GetIP(flag.Name)
		return val
	case "ipMask":
		val, _ := set.GetIPv4Mask(flag.Name)
		return val
	case "ipNet":
		val, _ := set.GetIPNet(flag.Name)
		return val
	case "count":
		val, _ := set.GetCount(flag.Name)
		return val
	case "bytesHex":
		val, _ := set.GetBytesHex(flag.Name)
		return val
	case "bytesBase64":
		val, _ := set.GetBytesBase64(flag.Name)
		return val
	case "string":
		val, _ := set.GetString(flag.Name)
		return val
	case "stringSlice":
		val, _ := set.GetStringSlice(flag.Name)
		return val
	case "intSlice":
		val, _ := set.GetIntSlice(flag.Name)
		return val
	case "uintSlice":
		val, _ := set.GetUintSlice(flag.Name)
		return val
	case "int32Slice":
		val, _ := set.GetInt32Slice(flag.Name)
		return val
	case "int64Slice":
		val, _ := set.GetInt64Slice(flag.Name)
		return val
	case "float32Slice":
		val, _ := set.GetFloat32Slice(flag.Name)
		return val
	case "float64Slice":
		val, _ := set.GetFloat64Slice(flag.Name)
		return val
	case "boolSlice":
		val, _ := set.GetBoolSlice(flag.Name)
		return val
	case "durationSlice":
		val, _ := set.GetDurationSlice(flag.Name)
		return val
	case "ipSlice":
		val, _ := set.GetIPSlice(flag.Name)
		return val
	case "stringArray":
		val, _ := set.GetStringArray(flag.Name)
		return val
	case "stringToString":
		val, _ := set.GetStringToString(flag.Name)
		return val
	case "stringToInt":
		val, _ := set.GetStringToInt(flag.Name)
		return val
	case "stringToInt64":
		val, _ := set.GetStringToInt64(flag.Name)
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
