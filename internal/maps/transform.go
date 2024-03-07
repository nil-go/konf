// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

func TransformKeys(src map[string]interface{}, keyMap func(string) string) map[string]interface{} {
	if src == nil || keyMap == nil {
		return src
	}

	dst := make(map[string]interface{}, len(src))
	for key, value := range src {
		if m, ok := value.(map[string]interface{}); ok {
			value = TransformKeys(m, keyMap)
		}
		dst[keyMap(key)] = value
	}

	return dst
}
