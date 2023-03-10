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

func TestFlag_Load(t *testing.T) { //nolint:paralleltest
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

	for i := range testcases { //nolint:paralleltest
		testcase := testcases[i]

		t.Run(testcase.description, func(t *testing.T) {
			values, err := kflag.New(testcase.opts...).Load()
			require.NoError(t, err)
			require.Equal(t, testcase.expected, values)
		})
	}
}

func TestFlag_panic(t *testing.T) {
	t.Parallel()

	set := &flag.FlagSet{}
	set.Var(&panicker{doNotPanic: true}, "p", "")

	_, err := kflag.New(kflag.WithFlagSet(set)).Load()
	require.EqualError(t, err, "panic calling String method on zero flag_test.panicker for flag p: panic")
}

type panicker struct {
	flag.Value

	doNotPanic bool
}

func (p panicker) String() string {
	if p.doNotPanic {
		return ""
	}

	panic("panic")
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

			require.Equal(
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
