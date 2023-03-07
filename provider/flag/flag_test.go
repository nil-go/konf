// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package flag_test

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"

	kflag "github.com/ktong/konf/provider/flag"
)

func TestFlag(t *testing.T) {
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
			loader, err := kflag.New(testcase.opts...).Load()
			require.NoError(t, err)
			require.Equal(t, testcase.expected, loader)
		})
	}
}
