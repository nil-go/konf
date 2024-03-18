// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

//nolint:testpackage
package aws

import (
	"context"
	"embed"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/provider/appconfig"
	"github.com/nil-go/konf/provider/env"
	"github.com/nil-go/konf/provider/fs"
	"github.com/nil-go/konf/provider/s3"
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
	time.Sleep(20 * time.Second) // Wait for at lease two watch polls.

	fmt.Println()
	fmt.Println("konf.source:", config.Source)
	fmt.Println()
	fmt.Println(konf.Explain("konf.source"))
	// Output:
	// load executed: loader=appconfig://konf/config.yaml, changed=false, error=<nil>
	// load executed: loader=s3://konf-test/config.yaml, changed=false, error=<nil>
	//
	// konf.source: AppConfig
	//
	// konf.source has value[AppConfig] that is loaded by loader[appconfig://konf/config.yaml].
	// Here are other value(loader)s:
	//   - S3(s3://konf-test/config.yaml)
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

	// Load configuration from AWS S3.
	if err := config.Load(s3.New(
		"s3://konf-test/config.yaml",
		s3.WithUnmarshal(yaml.Unmarshal),
		s3.WithPollInterval(15*time.Second),
	)); err != nil {
		panic(err) // handle error
	}
	// Load configuration from AWS AppConfig.
	if err := config.Load(appconfig.New(
		"konf", "test", "config.yaml",
		appconfig.WithUnmarshal(yaml.Unmarshal),
		appconfig.WithPollInterval(15*time.Second),
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
