// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assert

import (
	"reflect"
	"testing"
)

func Equal[T any](tb testing.TB, expected, actual T) {
	tb.Helper()

	if !reflect.DeepEqual(expected, actual) {
		tb.Errorf("expected: %v; actual: %v", expected, actual)
	}
}

func NoError(tb testing.TB, err error) {
	tb.Helper()

	if err != nil {
		tb.Errorf("unexpected error: %v", err)
	}
}