// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"testing"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/internal/assert"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	config := konf.New()
	err := config.Load(mapLoader{"config": "string"})
	assert.NoError(t, err)
	konf.SetDefault(config)

	var v string
	assert.NoError(t, konf.Unmarshal("config", &v))
	assert.Equal(t, "string", v)
}

func TestGet(t *testing.T) {
	t.Parallel()

	config := konf.New()
	err := config.Load(mapLoader{"config": "string"})
	assert.NoError(t, err)
	konf.SetDefault(config)

	assert.Equal(t, "string", konf.Get[string]("config"))
}

func TestGet_error(t *testing.T) {
	buf := &buffer{}
	config := konf.New(konf.WithLogHandler(logHandler(buf)))
	err := config.Load(mapLoader{"config": "string"})
	assert.NoError(t, err)
	konf.SetDefault(config)

	assert.True(t, !konf.Get[bool]("config"))
	expected := `level=WARN msg="Could not read config, return empty value instead."` +
		` path=config type=bool` +
		` error="decode: cannot parse '' as bool: strconv.ParseBool: parsing \"string\": invalid syntax"` +
		"\n"
	assert.Equal(t, expected, buf.String())
}
