// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
)

func TestConfig_Exists(t *testing.T) {
	var config konf.Config
	assert.NoError(t, config.Load(mapLoader{"config": "string"}))
	assert.True(t, config.Exists([]string{"config"}))
	assert.True(t, !config.Exists([]string{"other"}))
}
