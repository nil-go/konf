// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env_test

import (
	"testing"

	"github.com/ktong/konf"
	"github.com/ktong/konf/internal/assert"
	"github.com/ktong/konf/provider/env"
)

var _ konf.Loader = (*env.Env)(nil)

func TestEnv_Load(t *testing.T) {
	testcases := []struct {
		description string
		opts        []env.Option
		expected    map[string]any
	}{
		{
			description: "with prefix",
			opts:        []env.Option{env.WithPrefix("P_")},
			expected: map[string]any{
				"P": map[string]any{
					"K": "v",
					"D": "-",
				},
			},
		},
		{
			description: "with delimiter",
			opts:        []env.Option{env.WithPrefix("P."), env.WithDelimiter(".")},
			expected: map[string]any{
				"P": map[string]any{
					"D": ".",
				},
			},
		},
	}

	t.Setenv("P_K", "v")
	t.Setenv("P_D", "-")
	t.Setenv("P.D", ".")

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			values, err := env.New(testcase.opts...).Load()
			assert.NoError(t, err)
			assert.Equal(t, testcase.expected, values)
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

			assert.Equal(t, testcase.expected, env.New(env.WithPrefix(testcase.prefix)).String())
		})
	}
}
