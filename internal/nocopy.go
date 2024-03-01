// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"reflect"
	"unsafe"
)

type NoCopy[T any] struct {
	addr *NoCopy[T] // of receiver, to detect copies by value
}

// Following code is copied from strings.Builder to avoid copying Config.

func (c *NoCopy[T]) Check() {
	if c.addr == nil {
		// This hack works around a failing of Go's escape analysis
		// that was causing c to escape and be heap allocated.
		// See issue 23382.
		// once issue 7921 is fixed, this should be reverted to just "c.addr = c".
		c.addr = (*NoCopy[T])(noescape(unsafe.Pointer(c)))
	} else if c.addr != c {
		panic("illegal use of non-zero " + reflect.TypeOf((*T)(nil)).Elem().Name() + " copied by value")
	}
}

// noescape hides a pointer from escape analysis. It is the identity function
// but escape analysis doesn't think the output depends on the input.
// noescape is inlined and currently compiles down to zero instructions.
// USE CAREFULLY!
// This was copied from the runtime; see issues 23382 and 7921.
//
//go:nosplit
//go:nocheckptr
func noescape(p unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) ^ 0) //nolint:staticcheck
}
