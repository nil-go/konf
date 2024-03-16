// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

import "strings"

func Sub(values map[string]any, path string, delimiter string) any {
	if path == "" {
		return values
	}

	key, path, _ := strings.Cut(path, delimiter)
	if path == "" {
		return values[key]
	}

	if mp, ok := values[key].(map[string]any); ok {
		return Sub(mp, path, delimiter)
	}

	return nil
}
