// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"embed"
	"fmt"

	"github.com/ktong/konf"
	"github.com/ktong/konf/provider/env"
	kfs "github.com/ktong/konf/provider/fs"
)

func ExampleGet() {
	ExampleSetGlobal()

	fmt.Print(konf.Get[string]("server.host"))
	// Output: example.com
}

func ExampleUnmarshal() {
	ExampleSetGlobal()

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

func ExampleSetGlobal() {
	cfg, err := konf.New(
		konf.WithLoader(
			kfs.New(testdata, "testdata/config.json"),
			env.New(env.WithPrefix("server")),
		),
	)
	if err != nil {
		// Handle error here.
		panic(err)
	}
	konf.SetGlobal(cfg)
	// Output:
}
