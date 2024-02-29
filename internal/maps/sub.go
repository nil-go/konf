// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

import "strings"

func Sub(values map[string]any, keys []string) any {
	if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
		return values
	}

	var next any = values
	for _, key := range keys {
		key = strings.ToLower(key)
		mp, ok := next.(map[string]any)
		if !ok {
			return nil
		}

		val, exist := mp[key]
		if !exist {
			return nil
		}
		next = val
	}

	return next
}
