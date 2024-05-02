// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

// Merge recursively merges the src map into the dst map.
// Key conflicts are resolved by preferring src,
// or recursively descending, if both values from src and dst are map.
func Merge(dst, src map[string]any, keyMap func(string) string) {
	for srcKey, srcVal := range src {
		dstKey := srcKey
		dstVal, ok := dst[dstKey]
		if !ok && keyMap != nil {
			mappedKey := keyMap(srcKey)
			for k, v := range dst {
				if keyMap(k) == mappedKey {
					dstKey = k
					dstVal = v

					break
				}
			}
		}

		// Direct override if the srcVal is not map[string]any.
		srcMap, srcOk := srcVal.(map[string]any)
		if !srcOk {
			delete(dst, dstKey)
			dst[srcKey] = srcVal

			continue
		}

		// Direct override if the dstVal is not map[string]any.
		dstMap, dstOk := dstVal.(map[string]any)
		if !dstOk {
			// Create a new map to avoid overwriting the src map.
			values := make(map[string]any)
			Merge(values, srcMap, keyMap)
			delete(dst, dstKey)
			dst[srcKey] = values

			continue
		}

		// Merge if the srcVal and dstVal are both map[string]any.
		Merge(dstMap, srcMap, keyMap)
	}
}
