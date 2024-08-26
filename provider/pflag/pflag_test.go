// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package pflag_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	kflag "github.com/nil-go/konf/provider/pflag"
	"github.com/nil-go/konf/provider/pflag/internal/assert"
)

func TestPFlag_empty(t *testing.T) {
	var loader kflag.PFlag
	values, err := loader.Load()
	assert.NoError(t, err)
	assert.Equal(t, "v", values["p"].(map[string]any)["k"].(string))
}

func TestPFlag_Load(t *testing.T) {
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
			opts: []kflag.Option{
				kflag.WithFlagSet(set),
				kflag.WithNameSplitter(func(s string) []string { return strings.Split(s, "_") }),
			},
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

	pflag.CommandLine.SortFlags = false
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			values, err := kflag.New(testcase.konf, testcase.opts...).Load()
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
			expected:    "pflag:P_*",
		},
		{
			description: "no prefix",
			expected:    "pflag:*",
		},
	}

	for _, testcase := range testcases {

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				testcase.expected,
				kflag.New(
					konfStub{},
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
	pflag.String("p_d", "_", "")
	pflag.Parse()

	set.String("k", "v", "")
}

type konfStub struct {
	exists bool
}

func (k konfStub) Exists([]string) bool {
	return k.exists
}
