// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package maps

type KeyValue struct {
	Key   string
	Value any
}

func Pack(key string, value any) KeyValue {
	return KeyValue{Key: key, Value: value}
}

func Unpack(value any) (string, any) {
	if v, ok := value.(KeyValue); ok {
		return v.Key, v.Value
	}

	return "", value
}
