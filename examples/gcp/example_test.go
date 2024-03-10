// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package gcp_test

import (
	"context"
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/provider/env"
	"github.com/nil-go/konf/provider/fs"
	"github.com/nil-go/konf/provider/gcs"
	"github.com/nil-go/konf/provider/secretmanager"
)

func Example() {
	// At the beginning of the application, load configuration from different sources.
	loadConfig()

	// ... 2,000 lines of code ...

	// Read the configuration.
	config := struct {
		Source string
	}{
		Source: "default",
	}
	if err := konf.Unmarshal("konf", &config); err != nil {
		panic(err) // handle error
	}
	konf.OnChange(func() {
		newConfig := config
		if err := konf.Unmarshal("konf", &newConfig); err != nil {
			panic(err) // handle error
		}
		config = newConfig
	})

	// This should not be part of the application. It's just for verification.
	fmt.Println(config.Source)
	fmt.Println()
	fmt.Println(konf.Explain("konf.source"))
	// Output:
	// Secret Manager
	//
	// konf.source has value[Secret Manager] that is loaded by loader[secret-manager://konf-test].
	// Here are other value(loader)s:
	//   - GCS(gs://konf-test/config.yaml)
	//   - Embedded FS(fs:///config/config.yaml)
}

func loadConfig() {
	var config konf.Config

	// Load configuration from embed file system.
	if err := config.Load(fs.New(configFS, "config/config.yaml", fs.WithUnmarshal(yaml.Unmarshal))); err != nil {
		panic(err) // handle error
	}
	// Load configuration from environment variables.
	if err := config.Load(env.New()); err != nil {
		panic(err) // handle error
	}

	// Load configuration from GCP Cloud Storage.
	if err := config.Load(gcs.New("gs://konf-test/config.yaml", gcs.WithUnmarshal(yaml.Unmarshal))); err != nil {
		panic(err) // handle error
	}
	// Load configuration from GCP Secret Manager.
	if err := config.Load(secretmanager.New(secretmanager.WithProject("konf-test"))); err != nil {
		panic(err) // handle error
	}

	// Watch the changes of configuration.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := config.Watch(ctx); err != nil {
			panic(err) // handle error
		}
	}()

	konf.SetDefault(&config)
}

//go:embed config
var configFS embed.FS
