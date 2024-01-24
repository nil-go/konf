// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package flag_test

import (
	"flag"
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
	kflag "github.com/nil-go/konf/provider/flag"
)

var _ konf.Loader = (*kflag.Flag)(nil)

func TestFlag_Load(t *testing.T) {
	flag.String("p.k", "", "")
	_ = flag.Set("p.k", "v")
	flag.String("p.d", ".", "")
	flag.Int("p.i", 0, "")

	set := &flag.FlagSet{}
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
			expected:    "flag:P_",
		},
		{
			description: "no prefix",
			expected:    "flag",
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
					kflag.WithFlagSet(&flag.FlagSet{}),
				).String(),
			)
		})
	}
}
