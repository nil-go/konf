// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

func Sub(values map[string]any, path []string, keyMap func(string) string) any {
	for len(path) > 0 && path[0] == "" {
		path = path[1:]
	}

	if len(path) == 0 {
		return values
	}

	key := path[0]
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

	if len(path) == 1 {
		return value
	}
	if mp, ok := value.(map[string]any); ok {
		return Sub(mp, path[1:], keyMap)
	}

	return nil
}
