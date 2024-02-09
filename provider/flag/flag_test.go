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

func TestFlag_New_panic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, r.(string), "cannot create Flag with nil konf")
		}
	}()
	kflag.New(nil)
}

func TestFlag_Load(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		exists      bool
		opts        []kflag.Option
		expected    map[string]any
	}{
		{
			description: "with flag set",
			opts:        []kflag.Option{kflag.WithFlagSet(set)},
			expected: map[string]any{
				"k": "v",
			},
		},
		{
			description: "with delimiter",
			opts: []kflag.Option{
				kflag.WithPrefix("p_"),
				kflag.WithNameSplitter(func(s string) []string {
					return strings.Split(s, "_")
				}),
			},
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

	parseOnce.Do(flag.Parse)
	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			values, err := kflag.New(konf{exists: testcase.exists}, testcase.opts...).Load()
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
					konf{},
					kflag.WithPrefix(testcase.prefix),
					kflag.WithFlagSet(&flag.FlagSet{}),
				).String(),
			)
		})
	}
}

var (
	parseOnce sync.Once
	set       = &flag.FlagSet{}
)

func init() {
	flag.String("p.k", "", "")
	_ = flag.Set("p.k", "v")
	flag.String("p.d", ".", "")
	flag.Int("p.i", 0, "")
	flag.String("p_d", "_", "")

	set.String("k", "v", "")
}
