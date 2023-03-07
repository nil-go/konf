// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf/internal/maps"
)

func TestInsert(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		description string
		keys        []string
		val         any
		dst         map[string]any
		expected    map[string]any
	}{
		{
			description: "empty",
			keys:        []string{"p", "k"},
			val:         "v",
			dst:         make(map[string]any),
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "override nested keys",
			keys:        []string{"p", "k"},
			val:         "v",
			dst: map[string]any{
				"p": map[string]any{
					"k": "a",
				},
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
		{
			description: "override non-map",
			keys:        []string{"p", "k"},
			val:         "v",
			dst: map[string]any{
				"p": "a",
			},
			expected: map[string]any{
				"p": map[string]any{
					"k": "v",
				},
			},
		},
	}

	for _, testcase := range testcases {
		testcase := testcase

		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()

			maps.Insert(testcase.dst, testcase.keys, testcase.val)
			require.Equal(t, testcase.expected, testcase.dst)
		})
	}
}
