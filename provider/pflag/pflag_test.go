// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package pflag_test

import (
	"flag"
	"testing"

	"github.com/spf13/pflag"

	kflag "github.com/nil-go/konf/provider/pflag"
	"github.com/nil-go/konf/provider/pflag/internal/assert"
)

func TestPFlag_New_panic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, r.(string), "cannot create Flag with nil konf")
		}
	}()
	kflag.New(nil)
}

func TestPFlag_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		exists      bool
		opts        []kflag.Option
		expected    map[string]any
	}{
		{
			description: "with flag set",
			opts:        []kflag.Option{kflag.WithFlagSet(set), kflag.WithDelimiter("_")},
			expected: map[string]any{
				"p": map[string]any{
					"d": "_",
				},
			},
		},
		{
			description: "with prefix",
			opts:        []kflag.Option{kflag.WithPrefix("p.")},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "with exists",
			exists:      true,
			opts:        []kflag.Option{kflag.WithPrefix("p.")},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
	}

	pflag.CommandLine.SortFlags = false
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			values, err := kflag.New(konf{exists: testcase.exists}, testcase.opts...).Load()
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, values)
		})
	}
}

func TestPFlag_String(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		prefix      string
		expected    string
	}{
		{
			description: "with prefix",
			prefix:      "P_",
			expected:    "pflag:P_",
		},
		{
			description: "no prefix",
			expected:    "pflag",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				testcase.expected,
				kflag.New(
					konf{},
					kflag.WithPrefix(testcase.prefix),
					kflag.WithFlagSet(&pflag.FlagSet{}),
				).String(),
			)
		})
	}
}

var set = &pflag.FlagSet{}

func init() {
	pflag.String("p.k", "", "")
	_ = pflag.Set("p.k", "v")
	pflag.String("p.d", ".", "")
	pflag.Int("p.i", 0, "")
	pflag.Parse()

	set.String("p_d", "_", "")
}
