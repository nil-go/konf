// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

// Insert recursively inserts the given value into the dst maps.
// Key conflicts are resolved by preferring the given value.
func Insert(dst map[string]any, keys []string, value any) {
	next := dst
	for _, key := range keys[:len(keys)-1] {
		val, exist := next[key]
		if !exist {
			// Create a map[string]any if the key does not exist.
			m := make(map[string]any)
			next[key] = m
			next = m

			continue
		}

		sub, ok := val.(map[string]any)
		if !ok {
			// Override if the val is not map[string]any.
			sub = make(map[string]any)
			next[key] = sub
		}
		next = sub
	}
	next[keys[len(keys)-1]] = value
}
