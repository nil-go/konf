// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"embed"
	"fmt"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/provider/env"
	kfs "github.com/nil-go/konf/provider/fs"
)

func ExampleGet() {
	ExampleSetDefault()

	fmt.Print(konf.Get[string]("server.host"))
	// Output: example.com
}

func ExampleUnmarshal() {
	ExampleSetDefault()

	cfg := struct {
		Host string
		Port int
	}{
		Host: "localhost",
		Port: 8080,
	}

	if err := konf.Unmarshal("server", &cfg); err != nil {
		// Handle error here.
		panic(err)
	}
	fmt.Printf("%s:%d\n", cfg.Host, cfg.Port)
	// Output: example.com:8080
}

//go:embed testdata
var testdata embed.FS

func ExampleSetDefault() {
	var config konf.Config
	if err := config.Load(kfs.New(testdata, "testdata/config.json")); err != nil {
		// Handle error here.
		panic(err)
	}
	if err := config.Load(env.New(env.WithPrefix("server"))); err != nil {
		// Handle error here.
		panic(err)
	}
	konf.SetDefault(&config)
	// Output:
}
