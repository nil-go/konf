// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assert

import "testing"

func EqualError(tb testing.TB, err error, message string) {
	tb.Helper()

	switch {
	case err == nil:
		tb.Errorf("\n  actual: <nil>\nexpected: %v", message)
	case err.Error() != message:
		tb.Errorf("\n  actual: %v\nexpected: %v", err.Error(), message)
	}
}
