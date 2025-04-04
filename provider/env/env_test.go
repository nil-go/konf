// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package env_test

import (
	"strings"
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/provider/env"
)

var _ konf.Loader = (*env.Env)(nil)

func TestEnv_empty(t *testing.T) {
	t.Setenv("P_K", "v")

	var loader env.Env
	values, err := loader.Load()
	assert.NoError(t, err)
	assert.Equal(t, "v", values["P"].(map[string]any)["K"].(string))
}

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
			opts: []env.Option{
				env.WithPrefix("P."),
				env.WithNameSplitter(func(s string) []string { return strings.Split(s, ".") }),
			},
			expected: map[string]any{
				"P": map[string]any{
					"D": ".",
				},
			},
		},
		{
			description: "with nil splitter",
			opts: []env.Option{
				env.WithPrefix("P."),
				env.WithNameSplitter(func(string) []string { return nil }),
			},
			expected: map[string]any{},
		},
		{
			description: "with empty splitter",
			opts: []env.Option{
				env.WithPrefix("P."),
				env.WithNameSplitter(func(string) []string { return []string{""} }),
			},
			expected: map[string]any{},
		},
	}

	t.Setenv("P_K", "v")
	t.Setenv("P_D", "-")
	t.Setenv("P.D", ".")
	t.Setenv("P.N", "")

	for _, testcase := range testcases {
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
			expected:    "env:P_*",
		},
		{
			description: "no prefix",
			expected:    "env:*",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testcase.expected, env.New(env.WithPrefix(testcase.prefix)).String())
		})
	}
}
