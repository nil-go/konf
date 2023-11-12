// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf_test

import (
	"context"
	"embed"
	"fmt"
	"sync"
	"time"

	"github.com/ktong/konf"
	"github.com/ktong/konf/provider/env"
	"github.com/ktong/konf/provider/file"
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

func ExampleWatch() {
	ExampleSetGlobal()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		if err := konf.Watch(ctx, func() {
			fmt.Print(konf.Get[string]("server.host"))
		}); err != nil {
			panic(err)
		}
	}()
	waitGroup.Wait()
	// Output:
}

//go:embed testdata
var testdata embed.FS

func ExampleSetGlobal() {
	cfg, err := konf.New(
		konf.WithLoader(
			file.New("testdata/config.json", file.WithFS(testdata)),
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
