// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps_test

import (
	"strings"
	"testing"

	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/internal/maps"
)

func TestTransformKeys(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description         string
		src                 map[string]any
		keyMap              func(string) string
		mapKeyCaseSensitive bool
		expected            map[string]any
	}{
		{
			description: "nil map",
			keyMap:      strings.ToLower,
		},
		{
			description: "nil keyMap",
			src:         map[string]any{"A": 1},
			expected:    map[string]any{"A": 1},
		},
		{
			description: "transform keys",
			src:         map[string]any{"A": map[string]any{"X": 1, "y": 2}},
			keyMap:      strings.ToLower,
			expected:    map[string]any{"a": map[string]any{"x": 1, "y": 2}},
		},
		{
			description:         "transform keys",
			src:                 map[string]any{"A": map[string]any{"X": 1, "y": 2}},
			keyMap:              strings.ToLower,
			mapKeyCaseSensitive: true,
			expected:            map[string]any{"a": maps.Pack("A", map[string]any{"x": maps.Pack("X", 1), "y": 2})},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			maps.TransformKeys(testcase.src, testcase.keyMap, testcase.mapKeyCaseSensitive)
			assert.Equal(t, testcase.expected, testcase.src)
		})
	}
}
