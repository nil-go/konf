// Copyright (c) 2024 The konf authors
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
		description string
		src         map[string]any
		keyMap      func(string) string
		expected    map[string]any
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
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			actual := maps.TransformKeys(tc.src, tc.keyMap)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
