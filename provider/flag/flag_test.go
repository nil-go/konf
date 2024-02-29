// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package flag_test

import (
	"flag"
	"strings"
	"sync"
	"testing"

	"github.com/nil-go/konf/internal/assert"
	kflag "github.com/nil-go/konf/provider/flag"
)

func TestFlag_empty(t *testing.T) {
	var loader kflag.Flag
	values, err := loader.Load()
	assert.NoError(t, err)
	assert.Equal(t, "v", values["p"].(map[string]any)["k"].(string))
}

func TestFlag_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		konf        *konfStub
		opts        []kflag.Option
		expected    map[string]any
	}{
		{
			description: "nil konf",
			opts:        []kflag.Option{kflag.WithPrefix("p.")},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
					"d": ".",
				},
			},
		},
		{
			description: "with flag set",
			konf:        &konfStub{exists: false},
			opts:        []kflag.Option{kflag.WithFlagSet(set)},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "with delimiter",
			konf:        &konfStub{exists: false},
			opts: []kflag.Option{
				kflag.WithPrefix("p_"),
				kflag.WithNameSplitter(func(s string) []string { return strings.Split(s, "_") }),
			},
			expected: map[string]any{
				"p": map[string]any{
					"d": "_",
				},
			},
		},
		{
			description: "with nil splitter",
			konf:        &konfStub{exists: false},
			opts: []kflag.Option{
				kflag.WithPrefix("p_"),
				kflag.WithNameSplitter(func(string) []string { return nil }),
			},
			expected: map[string]any{},
		},
		{
			description: "with empty splitter",
			konf:        &konfStub{exists: false},
			opts: []kflag.Option{
				kflag.WithPrefix("p_"),
				kflag.WithNameSplitter(func(string) []string { return []string{""} }),
			},
			expected: map[string]any{},
		},
		{
			description: "with prefix",
			konf:        &konfStub{exists: false},
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
			konf:        &konfStub{exists: true},
			opts:        []kflag.Option{kflag.WithPrefix("p.")},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
	}

	parse()
	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			values, err := kflag.New(testcase.konf, testcase.opts...).Load()
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

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				testcase.expected,
				kflag.New(
					konfStub{},
					kflag.WithPrefix(testcase.prefix),
					kflag.WithFlagSet(&flag.FlagSet{}),
				).String(),
			)
		})
	}
}

var (
	parse = sync.OnceFunc(flag.Parse)
	set   = &flag.FlagSet{}
)

func init() {
	flag.String("p.k", "", "")
	_ = flag.Set("p.k", "v")
	flag.String("p.d", ".", "")
	flag.Int("p.i", 0, "")
	flag.String("p_d", "_", "")

	set.String("k", "v", "")
}

type konfStub struct {
	exists bool
}

func (k konfStub) Exists([]string) bool {
	return k.exists
}
