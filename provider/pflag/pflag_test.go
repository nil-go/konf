// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package pflag_test

import (
	"testing"

	"github.com/spf13/pflag"

	kflag "github.com/nil-go/konf/provider/pflag"
	"github.com/nil-go/konf/provider/pflag/internal/assert"
)

func TestFlag_Load(t *testing.T) {
	pflag.String("p.k", "", "")
	_ = pflag.Set("p.k", "v")
	pflag.String("p.d", ".", "")
	pflag.Int("p.i", 0, "")

	set := &pflag.FlagSet{}
	set.String("p_d", "_", "")

	testcases := []struct {
		description string
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
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			values, err := kflag.New(testcase.opts...).Load()
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, values)
		})
	}
}

func TestFlag_String(t *testing.T) {
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
					kflag.WithPrefix(testcase.prefix),
					kflag.WithFlagSet(&pflag.FlagSet{}),
				).String(),
			)
		})
	}
}
