// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package azure_test

import (
	"context"
	"embed"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/provider/azappconfig"
	"github.com/nil-go/konf/provider/azblob"
	"github.com/nil-go/konf/provider/env"
	"github.com/nil-go/konf/provider/fs"
)

func Example() {
	// At the beginning of the application, load configuration from different sources.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loadConfig(ctx)

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
	time.Sleep(25 * time.Second) // Wait for at lease two watch polls.

	fmt.Println()
	fmt.Println("konf.source:", config.Source)
	fmt.Println()
	fmt.Println(konf.Explain("konf.source"))
	// Output:
	//
	// load executed: loader=https://konftest.blob.core.windows.net/konf-test/config.yaml, changed=false, error=<nil>
	// load executed: loader=https://konftest.azconfig.io, changed=false, error=<nil>
	//
	// konf.source: App Configuration
	//
	// konf.source has value[App Configuration] that is loaded by loader[https://konftest.azconfig.io].
	// Here are other value(loader)s:
	//   - Blob Storage(https://konftest.blob.core.windows.net/konf-test/config.yaml)
	//   - Embedded FS(fs:///config/config.yaml)
}

func loadConfig(ctx context.Context) {
	config := konf.New(konf.WithOnStatus(func(loader konf.Loader, changed bool, err error) {
		fmt.Printf("load executed: loader=%v, changed=%v, error=%v\n", loader, changed, err)
	}))

	// Load configuration from embed file system.
	if err := config.Load(fs.New(configFS, "config/config.yaml", fs.WithUnmarshal(yaml.Unmarshal))); err != nil {
		panic(err) // handle error
	}
	// Load configuration from environment variables.
	if err := config.Load(env.New()); err != nil {
		panic(err) // handle error
	}

	// Load configuration from Azure Blob Storage.
	if err := config.Load(azblob.New(
		"https://konftest.blob.core.windows.net", "konf-test", "config.yaml",
		azblob.WithUnmarshal(yaml.Unmarshal),
		azblob.WithPollInterval(15*time.Second),
	)); err != nil {
		panic(err) // handle error
	}
	// Load configuration from Azure App Configuration.
	if err := config.Load(azappconfig.New(
		"https://konftest.azconfig.io",
		azappconfig.WithPollInterval(20*time.Second),
	)); err != nil {
		panic(err) // handle error
	}

	// Watch the changes of configuration.
	go func() {
		if err := config.Watch(ctx); err != nil {
			panic(err) // handle error
		}
	}()

	konf.SetDefault(config)
}

//go:embed config
var configFS embed.FS
