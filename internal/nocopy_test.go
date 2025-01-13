// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal_test

import (
	"testing"

	"github.com/nil-go/konf/internal"
	"github.com/nil-go/konf/internal/assert"
)

func TestNoCopy(t *testing.T) {
	defer func() {
		assert.Equal(t, recover(), "illegal use of non-zero s copied by value")
	}()

	var s1 s
	s1.check()
	s2 := s1 //nolint:govet
	s2.check()

	t.Fail()
}

type s struct {
	nocopy internal.NoCopy[s]
}

func (s *s) check() {
	s.nocopy.Check()
}
