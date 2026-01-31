// Copyright (c) 2026 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assert

import (
	"reflect"
	"testing"
)

func Equal[T any](tb testing.TB, expected, actual T) {
	tb.Helper()

	if !reflect.DeepEqual(actual, expected) {
		tb.Errorf("\n  actual: %v\nexpected: %v", actual, expected)
	}
}

func NoError(tb testing.TB, err error) {
	tb.Helper()

	if err != nil {
		tb.Errorf("unexpected error: %v", err)
	}
}

func EqualError(tb testing.TB, err error, message string) {
	tb.Helper()

	switch {
	case err == nil:
		tb.Errorf("\n  actual: <nil>\nexpected: %v", message)
	case err.Error() != message:
		tb.Errorf("\n  actual: %v\nexpected: %v", err.Error(), message)
	}
}

func True(tb testing.TB, value bool) {
	tb.Helper()

	if !value {
		tb.Errorf("expected True")
	}
}
