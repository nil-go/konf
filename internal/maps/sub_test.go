// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps_test

import (
	"testing"

	"github.com/nil-go/konf/internal/assert"
	"github.com/nil-go/konf/internal/maps"
)

func TestSub(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		values      map[string]any
		path        []string
		expected    any
	}{
		{
			description: "nil values",
			values:      nil,
			path:        []string{"a", "b"},
			expected:    nil,
		},
		{
			description: "empty values",
			values:      map[string]any{},
			path:        []string{"a", "b"},
			expected:    nil,
		},
		{
			description: "empty keys",
			values:      map[string]any{"a": 1},
			path:        nil,
			expected:    map[string]any{"a": 1},
		},
		{
			description: "blank keys",
			values:      map[string]any{"a": 1},
			path:        []string{""},
			expected:    map[string]any{"a": 1},
		},
		{
			description: "lower case keys",
			values:      map[string]any{"a": 1, "b": 2},
			path:        []string{"a"},
			expected:    1,
		},
		{
			description: "upper case keys",
			values:      map[string]any{"A": 1},
			path:        []string{"A"},
			expected:    1,
		},
		{
			description: "value not exist",
			values:      map[string]any{"a": 1},
			path:        []string{"a", "b"},
			expected:    nil,
		},
		{
			description: "nest map",
			values:      map[string]any{"a": map[string]any{"x": 1, "y": 2}},
			path:        []string{"a", "y"},
			expected:    2,
		},
		{
			description: "non-map value",
			values:      map[string]any{"a": map[string]any{"x": 1}},
			path:        []string{"x", "y"},
			expected:    nil,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			actual := maps.Sub(testcase.values, testcase.path, nil)
			assert.Equal(t, testcase.expected, actual)
		})
	}
}
