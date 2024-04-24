// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package aws_test

import (
	"context"
	"embed"
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/notifier/sns"
	"github.com/nil-go/konf/provider/appconfig"
	"github.com/nil-go/konf/provider/env"
	"github.com/nil-go/konf/provider/fs"
	"github.com/nil-go/konf/provider/parameterstore"
	"github.com/nil-go/konf/provider/s3"
)

func Example() {
	// At the beginning of the application, load configuration from different sources.
	ctx, cancel := context.WithCancel(context.Background())
	wait := loadConfig(ctx)
	defer func() {
		cancel()
		wait()
	}()

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
	time.Sleep(42 * time.Second) // Wait for at lease two watch polls.

	fmt.Println()
	fmt.Println("konf.source:", config.Source)
	fmt.Println()
	fmt.Println(konf.Explain("konf.source"))
	// Output:
	// load executed: loader=s3://konf-test/config.yaml, changed=false, error=<nil>
	// load executed: loader=appconfig://konf/config.yaml, changed=false, error=<nil>
	// load executed: loader=parameter-store:/, changed=false, error=<nil>
	// load executed: loader=s3://konf-test/config.yaml, changed=false, error=<nil>
	// load executed: loader=appconfig://konf/config.yaml, changed=false, error=<nil>
	// load executed: loader=parameter-store:/, changed=false, error=<nil>
	//
	// konf.source: Parameter Store
	//
	// konf.source has value[Parameter Store] that is loaded by loader[parameter-store:/].
	// Here are other value(loader)s:
	//   - AppConfig(appconfig://konf/config.yaml)
	//   - S3(s3://konf-test/config.yaml)
	//   - Embedded FS(fs:///config/config.yaml)
}

func loadConfig(ctx context.Context) func() {
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
	s3Loader := s3.New(
		"s3://konf-test/config.yaml",
		s3.WithUnmarshal(yaml.Unmarshal),
		s3.WithPollInterval(15*time.Second),
	)
	if err := config.Load(s3Loader); err != nil {
		panic(err) // handle error
	}
	// Load configuration from AWS AppConfig.
	appConfigLoader := appconfig.New(
		"konf", "test", "config.yaml",
		appconfig.WithUnmarshal(yaml.Unmarshal),
		appconfig.WithPollInterval(18*time.Second),
	)
	if err := config.Load(appConfigLoader); err != nil {
		panic(err) // handle error
	}
	parameterStoreLoader := parameterstore.New(parameterstore.WithPollInterval(20 * time.Second))
	if err := config.Load(parameterStoreLoader); err != nil {
		panic(err) // handle error
	}
	konf.SetDefault(config)

	// Watch the changes of configuration.
	go func() {
		if err := config.Watch(ctx); err != nil {
			panic(err) // handle error
		}
	}()

	// Notify the changes of configuration.
	notifier := sns.NewNotifier("konf-test")
	notifier.Register(s3Loader, appConfigLoader, parameterStoreLoader)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		if err := notifier.Start(ctx); err != nil {
			panic(err) // handle error
		}
	}()

	return func() {
		waitGroup.Wait()
	}
}

//go:embed config
var configFS embed.FS
