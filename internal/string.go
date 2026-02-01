// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import "unsafe"

func String2ByteSlice(str string) []byte {
	if str == "" {
		return nil
	}

	return unsafe.Slice(unsafe.StringData(str), len(str))
}

func ByteSlice2String(bs []byte) string {
	if len(bs) == 0 {
		return ""
	}

	return unsafe.String(unsafe.SliceData(bs), len(bs))
}
