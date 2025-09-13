// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assert

import (
	"reflect"
	"strings"
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

func EqualContains(tb testing.TB, err error, message string) {
	tb.Helper()

	switch {
	case err == nil:
		tb.Errorf("\n  actual: <nil>\nexpected: %v", message)
	case !strings.Contains(err.Error(), message):
		tb.Errorf("\n  %v\n does not contains %v", err.Error(), message)
	}
}
