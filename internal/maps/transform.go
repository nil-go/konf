// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

func TransformKeys(src map[string]interface{}, keyMap func(string) string, mapKeyCaseSensitive bool) {
	if src == nil || keyMap == nil {
		return
	}
	for key, value := range src {
		if m, ok := value.(map[string]interface{}); ok {
			TransformKeys(m, keyMap, mapKeyCaseSensitive)
		}
		newKey := keyMap(key)
		if newKey != key {
			delete(src, key)
			if mapKeyCaseSensitive {
				src[newKey] = Pack(key, value)
			} else {
				src[newKey] = value
			}
		}
	}
}
