// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

import "slices"

func Sub(values map[string]any, path []string) any {
	path = slices.Compact(path)
	if len(path) == 0 {
		return values
	}

	_, value := Unpack(values[path[0]])
	if len(path) == 1 {
		return value
	}

	if mp, ok := value.(map[string]any); ok {
		return Sub(mp, path[1:])
	}

	return nil
}
