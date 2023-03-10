// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf/provider/env"
)

func TestEnv_Load(t *testing.T) { //nolint:paralleltest
	testcases := []struct {
		description string
		opts        []env.Option
		expected    map[string]any
	}{
		{
			description: "with prefix",
			opts:        []env.Option{env.WithPrefix("P_")},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": "-",
				},
			},
		},
		{
			description: "with delimiter",
			opts:        []env.Option{env.WithPrefix("P."), env.WithDelimiter(".")},
			expected: map[string]any{
				"p": map[string]any{
					"d": ".",
				},
			},
		},
	}

	t.Setenv("P_K", "v")
	t.Setenv("P_D", "-")
	t.Setenv("P.D", ".")

	for i := range testcases { //nolint:paralleltest
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			values, err := env.New(testcase.opts...).Load()
			require.NoError(t, err)
			require.Equal(t, testcase.expected, values)
		})
	}
}

func TestEnv_String(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		prefix      string
		expected    string
	}{
		{
			description: "with prefix",
			prefix:      "P_",
			expected:    "env:P_",
		},
		{
			description: "no prefix",
			expected:    "env",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testcase.expected, env.New(env.WithPrefix(testcase.prefix)).String())
		})
	}
}
