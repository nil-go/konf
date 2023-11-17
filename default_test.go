// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//go:build !race

package konf_test

import (
	"bytes"
	"log"
	"testing"

	"github.com/ktong/konf"
	"github.com/ktong/konf/internal/assert"
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
	config := konf.New()
	err := config.Load(mapLoader{"config": "string"})
	assert.NoError(t, err)
	konf.SetDefault(config)

	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	log.SetFlags(0)

	assert.True(t, !konf.Get[bool]("config"))
	expected := "ERROR Could not read config, return empty value instead." +
		" error=\"decode: cannot parse '' as bool: strconv.ParseBool: parsing \\\"string\\\": invalid syntax\"" +
		" path=config type=bool\n"
	assert.Equal(t, expected, buf.String())
}
