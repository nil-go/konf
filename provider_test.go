// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ktong/konf"
)

func TestConfig_Load(t *testing.T) {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	log.SetFlags(0)

	cfg := konf.New()
	require.NoError(t, cfg.Load(mapLoader{}))

	require.Equal(t, "Info Loaded configuration. loader=map[]\n", buf.String())
}

func TestConfig_nil(t *testing.T) {
	cfg := konf.New()
	require.NoError(t, cfg.Load(nil))
}

func TestConfig_error(t *testing.T) {
	cfg := konf.New()
	require.EqualError(t, cfg.Load(loader{}), "load configuration: error")
}

type mapLoader map[string]any

func (m mapLoader) Load() (map[string]any, error) {
	return m, nil
}

type loader struct{}

func (loader) Load() (map[string]any, error) {
	return nil, errors.New("error")
}
