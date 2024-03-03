// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"reflect"
)

type NoCopy[T any] struct {
	addr *NoCopy[T] // of receiver, to detect copies by value
}

func (c *NoCopy[T]) Check() {
	if c.addr == nil {
		c.addr = c
	}

	if c.addr != c {
		panic("illegal use of non-zero " + reflect.TypeOf((*T)(nil)).Elem().Name() + " copied by value")
	}
}
