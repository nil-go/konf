// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package convert_test

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/internal/convert"
	"github.com/nil-go/konf/internal/maps"
)

func TestConverter(t *testing.T) { //nolint:maintidx
	t.Parallel()

	testcases := []struct {
		description string
		opts        []convert.Option
		from        any
		to          any
		expected    any
		err         string
	}{
		// nil to.
		{
			description: "to is nil",
			err:         "to must be a pointer",
		},
		{
			description: "to is not a pointer",
			to:          struct{}{},
			err:         "to must be a pointer",
		},
		{
			description: "to is not an addressable pointer",
			from:        "str",
			to:          (*string)(nil),
			err:         "to must be addressable (a pointer)",
		},
		// nil from.
		{
			description: "from is nil",
			from:        nil,
			to:          pointer("str"),
			expected:    pointer("str"),
		},
		{
			description: "from is typed nil",
			from:        (*string)(nil),
			to:          pointer("str"),
			expected:    pointer("str"),
		},
		{
			description: "from is nested typed nil",
			from:        pointer(pointer((*string)(nil))),
			to:          pointer("str"),
			expected:    pointer("str"),
		},
		// Hook.
		{
			description: "string to []string",
			opts: []convert.Option{
				convert.WithHook[string, []string](func(f string) ([]string, error) {
					return strings.Split(f, ","), nil
				}),
			},
			from:     "a,b,c",
			to:       pointer([]string(nil)),
			expected: pointer([]string{"a", "b", "c"}),
		},
		{
			description: "string to duration",
			opts: []convert.Option{
				convert.WithHook[string, time.Duration](time.ParseDuration),
			},
			from:     "2s",
			to:       pointer(time.Duration(0)),
			expected: pointer(2 * time.Second),
		},
		{
			description: "string to duration (with unsupported hook)",
			opts: []convert.Option{
				convert.WithHook[any, any](func(any, any) error {
					return errors.ErrUnsupported
				}),
				convert.WithHook[string, time.Duration](time.ParseDuration),
			},
			from:     "2s",
			to:       pointer(time.Duration(0)),
			expected: pointer(2 * time.Second),
		},
		{
			description: "string to duration (in array)",
			opts: []convert.Option{
				convert.WithHook[string, time.Duration](time.ParseDuration),
			},
			from:     [1]string{"2s"},
			to:       pointer([1]time.Duration{0}),
			expected: pointer([1]time.Duration{2 * time.Second}),
		},
		{
			description: "string to duration (pointer in array)",
			opts: []convert.Option{
				convert.WithHook[string, time.Duration](time.ParseDuration),
			},
			from:     [1]string{"2s"},
			to:       pointer([1]*time.Duration{nil}),
			expected: pointer([1]*time.Duration{pointer(2 * time.Second)}),
		},
		{
			description: "string to duration (in slice)",
			opts: []convert.Option{
				convert.WithHook[string, time.Duration](time.ParseDuration),
			},
			from:     []string{"2s"},
			to:       pointer([]time.Duration(nil)),
			expected: pointer([]time.Duration{2 * time.Second}),
		},
		{
			description: "string to duration (pointer in slice)",
			opts: []convert.Option{
				convert.WithHook[string, time.Duration](time.ParseDuration),
			},
			from:     []string{"2s"},
			to:       pointer([]*time.Duration(nil)),
			expected: pointer([]*time.Duration{pointer(2 * time.Second)}),
		},
		{
			description: "text unmarshaler",
			opts: []convert.Option{
				convert.WithHook[string, encoding.TextUnmarshaler](func(f string, t encoding.TextUnmarshaler) error {
					return t.UnmarshalText([]byte(f))
				}),
			},
			from:     "sky",
			to:       pointer(Unknown),
			expected: pointer(Sky),
		},
		// To bool.
		{
			description: "bool to bool",
			from:        true,
			to:          pointer(false),
			expected:    pointer(true),
		},
		{
			description: "bool to bool (pointer)",
			from:        pointer(true),
			to:          pointer(false),
			expected:    pointer(true),
		},
		{
			description: "int to bool (false)",
			from:        0,
			to:          pointer(true),
			expected:    pointer(false),
		},
		{
			description: "int to bool (true)",
			from:        42,
			to:          pointer(false),
			expected:    pointer(true),
		},
		{
			description: "uint to bool (false)",
			from:        uint(0),
			to:          pointer(true),
			expected:    pointer(false),
		},
		{
			description: "uint to bool (true)",
			from:        uint(1),
			to:          pointer(false),
			expected:    pointer(true),
		},
		{
			description: "float to bool (false)",
			from:        float32(0),
			to:          pointer(true),
			expected:    pointer(false),
		},
		{
			description: "float to bool (true)",
			from:        float32(1),
			to:          pointer(false),
			expected:    pointer(true),
		},
		{
			description: "complex to bool (false)",
			from:        complex(0, 0),
			to:          pointer(true),
			expected:    pointer(false),
		},
		{
			description: "complex to bool (true)",
			from:        complex(0, 1),
			to:          pointer(false),
			expected:    pointer(true),
		},
		{
			description: "string to bool (false)",
			from:        "F",
			to:          pointer(true),
			expected:    pointer(false),
		},
		{
			description: "string to bool (true)",
			from:        "T",
			to:          pointer(false),
			expected:    pointer(true),
		},
		{
			description: "string to bool (empty)",
			from:        "",
			to:          pointer(true),
			expected:    pointer(false),
		},
		{
			description: "string to bool (non-empty)",
			from:        "str",
			to:          pointer(false),
			err:         "cannot parse '' as bool: strconv.ParseBool: parsing \"str\": invalid syntax",
		},
		{
			description: "unsupported type to bool (non-empty)",
			from:        []string{"str"},
			to:          pointer(false),
			err:         "'' expected type 'bool', got unconvertible type '[]string', value: '[str]'",
		},
		// To int.
		{
			description: "bool to int (false)",
			from:        false,
			to:          pointer(1),
			expected:    pointer(0),
		},
		{
			description: "bool to int (true)",
			from:        true,
			to:          pointer(0),
			expected:    pointer(1),
		},
		{
			description: "bool to int (pointer)",
			from:        pointer(true),
			to:          pointer(0),
			expected:    pointer(1),
		},
		{
			description: "int to int",
			from:        42,
			to:          pointer(0),
			expected:    pointer(42),
		},
		{
			description: "uint to int",
			from:        uint(42),
			to:          pointer(0),
			expected:    pointer(42),
		},
		{
			description: "float to int",
			from:        float32(42),
			to:          pointer(0),
			expected:    pointer(42),
		},
		{
			description: "complex to int",
			from:        complex(42, 1),
			to:          pointer(0),
			expected:    pointer(42),
		},
		{
			description: "string to int",
			from:        "42",
			to:          pointer(0),
			expected:    pointer(42),
		},
		{
			description: "string to int (empty)",
			from:        "",
			to:          pointer(42),
			expected:    pointer(0),
		},
		{
			description: "string to int (non-number)",
			from:        "str",
			to:          pointer(0),
			err:         "cannot parse '' as int: strconv.ParseInt: parsing \"str\": invalid syntax",
		},
		{
			description: "json number to int",
			from:        json.Number("42"),
			to:          pointer(0),
			expected:    pointer(42),
		},
		{
			description: "json number  to int (empty)",
			from:        json.Number(""),
			to:          pointer(42),
			expected:    pointer(0),
		},
		{
			description: "json number  to int (non-number)",
			from:        json.Number("str"),
			to:          pointer(0),
			err:         "cannot parse '' as int: strconv.ParseInt: parsing \"str\": invalid syntax",
		},
		{
			description: "unsupported type to int (non-empty)",
			from:        []string{"str"},
			to:          pointer(0),
			err:         "'' expected type 'int', got unconvertible type '[]string', value: '[str]'",
		},
		// To uint.
		{
			description: "bool to uint (false)",
			from:        false,
			to:          pointer(uint(1)),
			expected:    pointer(uint(0)),
		},
		{
			description: "bool to uint (true)",
			from:        true,
			to:          pointer(uint(0)),
			expected:    pointer(uint(1)),
		},
		{
			description: "bool to uint (pointer)",
			from:        pointer(true),
			to:          pointer(uint(0)),
			expected:    pointer(uint(1)),
		},
		{
			description: "int to uint",
			from:        42,
			to:          pointer(uint(0)),
			expected:    pointer(uint(42)),
		},
		{
			description: "int to uint (negative)",
			from:        -42,
			to:          pointer(uint(0)),
			err:         "cannot parse '', -42 overflows uint",
		},
		{
			description: "uint to uint",
			from:        uint(42),
			to:          pointer(uint(0)),
			expected:    pointer(uint(42)),
		},
		{
			description: "float to uint",
			from:        float32(42),
			to:          pointer(uint(0)),
			expected:    pointer(uint(42)),
		},
		{
			description: "float to uint (negative)",
			from:        float32(-42),
			to:          pointer(uint(0)),
			err:         "cannot parse '', -42.000000 overflows uint",
		},
		{
			description: "complex to uint",
			from:        complex(42, 1),
			to:          pointer(uint(0)),
			expected:    pointer(uint(42)),
		},
		{
			description: "complex to uint (negative)",
			from:        complex(-42, 1),
			to:          pointer(uint(0)),
			err:         "cannot parse '', -42.000000 overflows uint",
		},
		{
			description: "string to uint",
			from:        "42",
			to:          pointer(uint(0)),
			expected:    pointer(uint(42)),
		},
		{
			description: "string to uint (empty)",
			from:        "",
			to:          pointer(uint(42)),
			expected:    pointer(uint(0)),
		},
		{
			description: "string to uint (non-number)",
			from:        "str",
			to:          pointer(uint(0)),
			err:         "cannot parse '' as uint: strconv.ParseUint: parsing \"str\": invalid syntax",
		},
		{
			description: "json number to uint",
			from:        json.Number("42"),
			to:          pointer(uint(0)),
			expected:    pointer(uint(42)),
		},
		{
			description: "json number  to uint (empty)",
			from:        json.Number(""),
			to:          pointer(uint(42)),
			expected:    pointer(uint(0)),
		},
		{
			description: "json number  to uint (non-number)",
			from:        json.Number("str"),
			to:          pointer(uint(0)),
			err:         "cannot parse '' as uint: strconv.ParseUint: parsing \"str\": invalid syntax",
		},
		{
			description: "unsupported type to uint (non-empty)",
			from:        []string{"str"},
			to:          pointer(uint(0)),
			err:         "'' expected type 'uint', got unconvertible type '[]string', value: '[str]'",
		},
		// To float.
		{
			description: "bool to float (false)",
			from:        false,
			to:          pointer(float64(1)),
			expected:    pointer(float64(0)),
		},
		{
			description: "bool to float (true)",
			from:        true,
			to:          pointer(float64(0)),
			expected:    pointer(float64(1)),
		},
		{
			description: "bool to float (pointer)",
			from:        pointer(true),
			to:          pointer(float64(0)),
			expected:    pointer(float64(1)),
		},
		{
			description: "int to float",
			from:        42,
			to:          pointer(float64(0)),
			expected:    pointer(float64(42)),
		},
		{
			description: "uint to float",
			from:        uint(42),
			to:          pointer(float64(0)),
			expected:    pointer(float64(42)),
		},
		{
			description: "float to float",
			from:        float32(42),
			to:          pointer(float64(0)),
			expected:    pointer(float64(42)),
		},
		{
			description: "complex to float",
			from:        complex(42, 1),
			to:          pointer(float64(0)),
			expected:    pointer(float64(42)),
		},
		{
			description: "string to float",
			from:        "42",
			to:          pointer(float64(0)),
			expected:    pointer(float64(42)),
		},
		{
			description: "string to float (empty)",
			from:        "",
			to:          pointer(float64(42)),
			expected:    pointer(float64(0)),
		},
		{
			description: "string to float (non-number)",
			from:        "str",
			to:          pointer(float64(0)),
			err:         "cannot parse '' as float: strconv.ParseFloat: parsing \"str\": invalid syntax",
		},
		{
			description: "json number to float",
			from:        json.Number("42"),
			to:          pointer(float64(0)),
			expected:    pointer(float64(42)),
		},
		{
			description: "json number  to float (empty)",
			from:        json.Number(""),
			to:          pointer(float64(42)),
			expected:    pointer(float64(0)),
		},
		{
			description: "json number  to float (non-number)",
			from:        json.Number("str"),
			to:          pointer(float64(0)),
			err:         "cannot parse '' as float: strconv.ParseFloat: parsing \"str\": invalid syntax",
		},
		{
			description: "unsupported type to float (non-empty)",
			from:        []string{"str"},
			to:          pointer(float64(0)),
			err:         "'' expected type 'float64', got unconvertible type '[]string', value: '[str]'",
		},
		// To complex.
		{
			description: "bool to complex (false)",
			from:        false,
			to:          pointer(complex(1, 0)),
			expected:    pointer(complex(0, 0)),
		},
		{
			description: "bool to complex (true)",
			from:        true,
			to:          pointer(complex(0, 0)),
			expected:    pointer(complex(1, 0)),
		},
		{
			description: "bool to complex (pointer)",
			from:        pointer(true),
			to:          pointer(complex(0, 0)),
			expected:    pointer(complex(1, 0)),
		},
		{
			description: "int to complex",
			from:        42,
			to:          pointer(complex(0, 0)),
			expected:    pointer(complex(42, 0)),
		},
		{
			description: "uint to complex",
			from:        uint(42),
			to:          pointer(complex(0, 0)),
			expected:    pointer(complex(42, 0)),
		},
		{
			description: "float to complex",
			from:        float32(42),
			to:          pointer(complex(0, 0)),
			expected:    pointer(complex(42, 0)),
		},
		{
			description: "complex to complex",
			from:        complex(42, 1),
			to:          pointer(complex(0, 0)),
			expected:    pointer(complex(42, 1)),
		},
		{
			description: "string to complex",
			from:        "42+1i",
			to:          pointer(complex(0, 0)),
			expected:    pointer(complex(42, 1)),
		},
		{
			description: "string to complex (empty)",
			from:        "",
			to:          pointer(complex(42, 0)),
			expected:    pointer(complex(0, 0)),
		},
		{
			description: "string to complex (non-number)",
			from:        "str",
			to:          pointer(complex(0, 0)),
			err:         "cannot parse '' as complex: strconv.ParseComplex: parsing \"str\": invalid syntax",
		},
		{
			description: "unsupported type to complex (non-empty)",
			from:        []string{"str"},
			to:          pointer(complex(0, 0)),
			err:         "'' expected type 'complex128', got unconvertible type '[]string', value: '[str]'",
		},
		// To array.
		{
			description: "array to array",
			from:        [3]string{"1", "2", "3"},
			to:          pointer([3]int{3, 2, 1}),
			expected:    pointer([3]int{1, 2, 3}),
		},
		{
			description: "slice to array",
			from:        []string{"1", "2"},
			to:          pointer([3]int{3, 2, 1}),
			expected:    pointer([3]int{1, 2}),
		},
		{
			description: "slice to array (too big)",
			from:        make([]string, 2),
			to:          pointer([1]string{}),
			err:         "'': expected source data to have length less or equal to 1, got 2",
		},
		{
			description: "slice to array (element convert error)",
			from:        []int{-42, -43},
			to:          pointer([2]uint{}),
			err:         "cannot parse '[0]', -42 overflows uint\ncannot parse '[1]', -43 overflows uint",
		},
		{
			description: "empty map to array",
			from:        map[string]string{},
			to:          pointer([1]string{"str"}),
			expected:    pointer([1]string{}),
		},
		{
			description: "non-empty map to array",
			from:        map[string]string{"OuterField": "v"},
			to:          pointer([1]OuterStruct{}),
			expected:    pointer([1]OuterStruct{{OuterField: "v"}}),
		},
		{
			description: "int to array",
			from:        42,
			to:          pointer([1]int{}),
			expected:    pointer([1]int{42}),
		},
		// To map.
		{
			description: "nil map to map",
			from:        map[string]string(nil),
			to:          pointer(map[string]string{"k": "v"}),
			expected:    pointer(map[string]string(nil)),
		},
		{
			description: "empty map to map",
			from:        map[string]string{},
			to:          pointer(map[string]string{"k": "v"}),
			expected:    pointer(map[string]string{}),
		},
		{
			description: "non-empty map to map",
			from:        map[string]string{"k": "v"},
			to:          pointer(map[string][]byte(nil)),
			expected:    pointer(map[string][]byte{"k": []byte("v")}),
		},
		{
			description: "non-empty map to map (pointer)",
			from:        map[string]string{"2": "42"},
			to:          pointer(map[int]*int{0: pointer(40), 1: pointer(1)}),
			expected:    pointer(map[int]*int{2: pointer(42)}),
		},
		{
			description: "unsupported type to map",
			from:        "str",
			to:          pointer(map[string]string(nil)),
			err:         "'' expected a map, got 'string'",
		},
		{
			description: "map to map (key convert error)",
			from:        map[string]int{"-2": 42},
			to:          pointer(map[uint]uint(nil)),
			err:         "cannot parse '[-2]' as uint: strconv.ParseUint: parsing \"-2\": invalid syntax",
		},
		{
			description: "map to map (value convert error)",
			from:        map[string]int{"2": -42},
			to:          pointer(map[uint]uint(nil)),
			err:         "cannot parse '[2]', -42 overflows uint",
		},
		{
			description: "slice to array (element convert error)",
			from:        []int{-42, -43},
			to:          pointer([2]uint{}),
			err:         "cannot parse '[0]', -42 overflows uint\ncannot parse '[1]', -43 overflows uint",
		},
		// To pointer.
		{
			description: "string to pointer",
			from:        "str",
			to:          pointer((*string)(nil)),
			expected:    pointer(pointer("str")),
		},
		{
			description: "nil chan to pointer",
			from:        chan int(nil),
			to:          pointer(pointer("str")),
			expected:    pointer((*string)(nil)),
		},
		{
			description: "nil func to pointer",
			from:        (func())(nil),
			to:          pointer(pointer("str")),
			expected:    pointer((*string)(nil)),
		},
		{
			description: "nil interface to pointer",
			from:        pointer(fmt.Stringer(nil)),
			to:          pointer(pointer("str")),
			expected:    pointer((*string)(nil)),
		},
		{
			description: "nil map to pointer",
			from:        map[string]string(nil),
			to:          pointer(pointer("str")),
			expected:    pointer((*string)(nil)),
		},
		{
			description: "nil slice to pointer",
			from:        []string(nil),
			to:          pointer(pointer("str")),
			expected:    pointer((*string)(nil)),
		},
		{
			description: "nil unsafe pointer to pointer",
			from:        unsafe.Pointer(nil),
			to:          pointer(pointer("str")),
			expected:    pointer((*string)(nil)),
		},
		// To slice.
		{
			description: "array to slice",
			from:        [2]string{"1", "2"},
			to:          pointer([]int{3, 2, 1}),
			expected:    pointer([]int{1, 2}),
		},
		{
			description: "slice to slice",
			from:        []string{"1", "2", "3"},
			to:          pointer([]int(nil)),
			expected:    pointer([]int{1, 2, 3}),
		},
		{
			description: "slice to slice (nil)",
			from:        []string(nil),
			to:          pointer([]int{1, 2, 3}),
			expected:    pointer([]int(nil)),
		},
		{
			description: "slice to slice (element convert error)",
			from:        []int{-42, -43},
			to:          pointer([]uint{}),
			err:         "cannot parse '[0]', -42 overflows uint\ncannot parse '[1]', -43 overflows uint",
		},
		{
			description: "empty map to slice",
			from:        map[string]string{},
			to:          pointer([]string{"str"}),
			expected:    pointer([]string(nil)),
		},
		{
			description: "non-empty map to slice",
			from:        map[string]string{"OuterField": "v"},
			to:          pointer([]OuterStruct{}),
			expected:    pointer([]OuterStruct{{OuterField: "v"}}),
		},
		{
			description: "int to slice",
			from:        42,
			to:          pointer([]int{}),
			expected:    pointer([]int{42}),
		},
		{
			description: "string to []byte",
			from:        "str",
			to:          pointer([]byte(nil)),
			expected:    pointer([]byte{'s', 't', 'r'}),
		},
		// To string.
		{
			description: "bool to string (false)",
			from:        false,
			to:          pointer("str"),
			expected:    pointer("0"),
		},
		{
			description: "bool to string (true)",
			from:        true,
			to:          pointer("str"),
			expected:    pointer("1"),
		},
		{
			description: "bool to string (pointer)",
			from:        pointer(true),
			to:          pointer("str"),
			expected:    pointer("1"),
		},
		{
			description: "int to string",
			from:        42,
			to:          pointer("str"),
			expected:    pointer("42"),
		},
		{
			description: "uint to string",
			from:        uint(42),
			to:          pointer("str"),
			expected:    pointer("42"),
		},
		{
			description: "float to string",
			from:        float32(42),
			to:          pointer("str"),
			expected:    pointer("42"),
		},
		{
			description: "complex to string",
			from:        complex(42, 1),
			to:          pointer("str"),
			expected:    pointer("(42+1i)"),
		},
		{
			description: "string to string",
			from:        "str",
			to:          pointer(""),
			expected:    pointer("str"),
		},
		{
			description: "byte array to string",
			from:        [3]byte{'s', 't', 'r'},
			to:          pointer(""),
			expected:    pointer("str"),
		},
		{
			description: "byte slice to string",
			from:        []byte{'s', 't', 'r'},
			to:          pointer(""),
			expected:    pointer("str"),
		},
		{
			description: "unsupported type to string (non-empty)",
			from:        []string{"str"},
			to:          pointer(""),
			err:         "'' expected type 'string', got unconvertible type '[]string', value: '[str]'",
		},
		// To struct.
		{
			description: "map to struct",
			opts: []convert.Option{
				convert.WithTagName("konf"),
				convert.WithHook[string, encoding.TextUnmarshaler](func(f string, t encoding.TextUnmarshaler) error {
					return t.UnmarshalText([]byte(f))
				}),
			},
			from: map[string]any{
				"Enum":           "sky",
				"OuterField":     "outer",
				"PrivateField":   "private",
				"InterfaceField": "interface{}",
				"InnerField":     "squash",
				"Inner":          map[string]any{"InnerField": "inner"},
			},
			to: pointer(OuterStruct{}),
			expected: pointer(OuterStruct{
				Enum:           Sky,
				OuterField:     "outer",
				InterfaceField: "interface{}",
				InnerStruct:    InnerStruct{InnerField: "squash"},
				Inner:          &InnerStruct{InnerField: "inner"},
			}),
		},
		{
			description: "map to struct (with keyMap)",
			opts: []convert.Option{
				convert.WithKeyMapper(strings.ToLower),
			},
			from: map[string]string{"innerfield": "inner", "interfacefield": "interface{}"},
			to: pointer(struct {
				InnerField     string
				InterfaceField interface{}
			}{}),
			expected: pointer(
				struct {
					InnerField     string
					InterfaceField interface{}
				}{
					InnerField:     "inner",
					InterfaceField: "interface{}",
				}),
		},
		{
			description: "convert error  on field",
			from:        map[string]int{"InnerField": -42},
			to: pointer(struct {
				InnerField uint
			}{}),
			err: "cannot parse 'InnerField', -42 overflows uint",
		},
		{
			description: "squash on field",
			opts: []convert.Option{
				convert.WithTagName("konf"),
			},
			from: map[string]string{},
			to: pointer(struct {
				InnerField string `konf:",squash"`
			}{}),
			err: "InnerField: unsupported type for squash: string",
		},
		{
			description: "unsupported key type to struct",
			from:        map[int]string{},
			to:          pointer(OuterStruct{}),
			err:         "'' needs a map with string keys, has 'int' keys",
		},
		{
			description: "unsupported type to struct",
			from:        "str",
			to:          pointer(OuterStruct{}),
			err:         "'' expected a map, got 'string'",
		},
		{
			description: "int to interface",
			from:        42,
			to:          pointer(any(nil)),
			expected:    pointer(any(42)),
		},
		{
			description: "string to interface",
			from:        "str",
			to:          pointer(any(nil)),
			expected:    pointer(any("str")),
		},
		{
			description: "float to interface",
			from:        42.42,
			to:          pointer(any(nil)),
			expected:    pointer(any(42.42)),
		},
		{
			description: "map to interface",
			from:        map[string]int{"key": 42, "keySensitive": 43},
			to:          pointer(any(nil)),
			expected:    pointer(any(map[string]int{"key": 42, "keySensitive": 43})),
		},
		{
			description: "map to interface (with keyMap)", // Probably redundant.
			opts: []convert.Option{
				convert.WithKeyMapper(strings.ToLower),
			},
			from:     map[string]int{"key": 42, "keysensitive": 43},
			to:       pointer(any(nil)),
			expected: pointer(any(map[string]int{"key": 42, "keysensitive": 43})),
		},
		{
			description: "packed KV and field to map[string]interface{}",
			from: map[string]interface{}{
				"key1": maps.KeyValue{
					Key:   "key1",
					Value: "value1",
				},
				"key2": "value2",
			},
			to: pointer(map[string]interface{}{}),
			expected: pointer(map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			}),
		},
		{
			description: "packed KV and field to struct (with keyMap)",
			opts: []convert.Option{
				convert.WithKeyMapper(strings.ToLower),
			},
			from: map[string]interface{}{
				"key1": maps.KeyValue{
					Key:   "key1",
					Value: "value1",
				},
				"key2": "value2",
				"key3": []int{1, 2},
			},
			to: pointer(struct {
				Key1 interface{}
				Key2 interface{}
				Key3 interface{}
			}{}),
			expected: pointer(struct {
				Key1 interface{}
				Key2 interface{}
				Key3 interface{}
			}{
				Key1: "value1",
				Key2: "value2",
				Key3: []int{1, 2},
			}),
		},
		{
			description: "packed KV and field to interface{}",
			from: map[string]interface{}{
				"key1": maps.KeyValue{
					Key:   "key1",
					Value: "value1",
				},
				"key2": "value2",
			},
			to: pointer(any(nil)),
			expected: pointer(any(map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			})),
		},
		{
			description: "nil map to interface{}",
			from:        map[string]interface{}(nil),
			to:          pointer(any(nil)),
			expected:    pointer(any(nil)),
		},
		{
			description: "slice to interface",
			from:        []int{1, 2, 3},
			to:          pointer(any(nil)),
			expected:    pointer(any([]int{1, 2, 3})),
		},
		// unsupported.
		{
			description: "to func (unsupported)",
			from:        "str",
			to:          pointer(func() {}),
			err:         ": unsupported type: func",
		},
		{
			description: "to chan (unsupported)",
			from:        "str",
			to:          pointer((chan int)(nil)),
			err:         ": unsupported type: chan",
		},
		{
			description: "to unsafe.Pointer (unsupported)",
			from:        "str",
			to:          pointer(unsafe.Pointer(nil)),
			err:         ": unsupported type: unsafe.Pointer",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			converter := convert.New(testcase.opts...)
			err := converter.Convert(testcase.from, testcase.to)
			if err != nil {
				assert.EqualError(t, err, testcase.err)
			} else {
				assert.Equal(t, testcase.expected, testcase.to)
			}
		})
	}
}

func pointer[T any](v T) *T { return &v }

type Enum int

const (
	Unknown Enum = iota
	Sky
	Land
)

func (e *Enum) UnmarshalText(text []byte) error {
	switch string(text) {
	case "sky":
		*e = Sky
	case "land":
		*e = Land
	default:
		*e = Unknown
	}

	return nil
}

type (
	OuterStruct struct {
		Enum           Enum
		OuterField     string
		privateField   string //nolint:unused
		InterfaceField interface{}

		InnerStruct `konf:",squash"`
		Inner       *InnerStruct
	}

	InnerStruct struct {
		InnerField string
	}
)
