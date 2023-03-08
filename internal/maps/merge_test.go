// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf/internal/maps"
)

func TestMerge(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		src         map[string]any
		dst         map[string]any
		expected    map[string]any
	}{
		{
			description: "nil source",
			src:         nil,
			dst:         make(map[string]any),
			expected:    make(map[string]any),
		},
		{
			description: "empty",
			src:         make(map[string]any),
			dst:         make(map[string]any),
			expected:    make(map[string]any),
		},
		{
			description: "no key conflict",
			src:         map[string]any{"b": 2},
			dst:         map[string]any{"a": 1},
			expected:    map[string]any{"a": 1, "b": 2},
		},
		{
			description: "key conflict",
			src:         map[string]any{"a": 0},
			dst:         map[string]any{"a": 1},
			expected:    map[string]any{"a": 0},
		},
		{
			description: "no key conflict (nest map)",
			src:         map[string]any{"a": map[string]any{"y": 2}},
			dst:         map[string]any{"a": map[string]any{"x": 1}},
			expected:    map[string]any{"a": map[string]any{"x": 1, "y": 2}},
		},
		{
			description: "key conflict (nest map)",
			src:         map[string]any{"a": map[string]any{"x": 2}},
			dst:         map[string]any{"a": map[string]any{"x": 1}},
			expected:    map[string]any{"a": map[string]any{"x": 2}},
		},
		{
			description: "key conflict (srcVal is not map)",
			src:         map[string]any{"a": 2},
			dst:         map[string]any{"a": map[string]any{"x": 1}},
			expected:    map[string]any{"a": 2},
		},
		{
			description: "key conflict (dstVal is not map)",
			src:         map[string]any{"a": map[string]any{"x": 2}},
			dst:         map[string]any{"a": 1},
			expected:    map[string]any{"a": map[string]any{"x": 2}},
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			maps.Merge(testcase.dst, testcase.src)
			require.Equal(t, testcase.expected, testcase.dst)
		})
	}
}
