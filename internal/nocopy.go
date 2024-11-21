// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"reflect"
	"sync/atomic"
)

type NoCopy[T any] struct {
	addr atomic.Pointer[NoCopy[T]] // of receiver, to detect copies by value
}

func (c *NoCopy[T]) Check() {
	if c.addr.CompareAndSwap(nil, c) {
		return
	}

	if c.addr.Load() != c {
		panic("illegal use of non-zero " + reflect.TypeFor[T]().Name() + " copied by value")
	}
}
