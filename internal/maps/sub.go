// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

import "strings"

func Sub(values map[string]any, path string, delimiter string, keyMap func(string) string) any {
	if path == "" {
		return values
	}

	key, path, _ := strings.Cut(path, delimiter)
	value, ok := values[key]
	if !ok && keyMap != nil {
		key = keyMap(key)
		for k, v := range values {
			k = keyMap(k)
			if k == key {
				value = v

				break
			}
		}
	}

	if path == "" {
		return value
	}
	if mp, ok := value.(map[string]any); ok {
		return Sub(mp, path, delimiter, keyMap)
	}

	return nil
}
