// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf"
)

func TestUnmarshal(t *testing.T) {
	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	require.NoError(t, err)
	konf.SetGlobal(cfg)

	var v string
	require.NoError(t, konf.Unmarshal("config", &v))
	require.Equal(t, "string", v)
}

func TestGet(t *testing.T) {
	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	require.NoError(t, err)
	konf.SetGlobal(cfg)

	require.Equal(t, "string", konf.Get[string]("config"))
}

func TestGet_error(t *testing.T) {
	cfg, err := konf.New(konf.WithLoader(mapLoader{"config": "string"}))
	require.NoError(t, err)
	konf.SetGlobal(cfg)

	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	log.SetFlags(0)

	require.False(t, konf.Get[bool]("config"))
	expected := "Error Could not read config, return empty value instead." +
		" error=[konf] decode: cannot parse '' as bool: strconv.ParseBool: parsing \"string\": invalid syntax" +
		" path=config type=bool\n"
	require.Equal(t, expected, buf.String())
}
