// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

// Merge recursively merges the src map into the dst map.
// Key conflicts are resolved by preferring src,
// or recursively descending, if both values from src and dst are map.
func Merge(dst, src map[string]any) {
	for key, srcVal := range src {
		// Direct override if the srcVal is not map[string]any.
		srcMap, srcOk := srcVal.(map[string]any)
		if !srcOk {
			dst[key] = srcVal

			continue
		}

		// Direct override if the dstVal is not map[string]any.
		dstMap, dstOk := dst[key].(map[string]any)
		if !dstOk {
			values := make(map[string]any)
			Merge(values, srcMap)
			dst[key] = values

			continue
		}

		// Merge if the srcVal and dstVal are both map[string]any.
		Merge(dstMap, srcMap)
	}
}
