//go:build integration

// Copyright (c) 2025 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package gcp_test

import (
	"context"
	"embed"
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nil-go/konf"
	"github.com/nil-go/konf/notifier/pubsub"
	"github.com/nil-go/konf/provider/env"
	"github.com/nil-go/konf/provider/fs"
	"github.com/nil-go/konf/provider/gcs"
	"github.com/nil-go/konf/provider/secretmanager"
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
	err := konf.Unmarshal("konf", &config)
	if err != nil {
		panic(err) // handle error
	}
	konf.OnChange(func() {
		newConfig := config
		err := konf.Unmarshal("konf", &newConfig)
		if err != nil {
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
	// load executed: loader=gs://konf-test/config.yaml, changed=false, error=<nil>
	// load executed: loader=secret-manager://konf-test, changed=false, error=<nil>
	// load executed: loader=gs://konf-test/config.yaml, changed=false, error=<nil>
	// load executed: loader=secret-manager://konf-test, changed=false, error=<nil>
	//
	// konf.source: Secret Manager
	//
	// konf.source has value[Secret Manager] that is loaded by loader[secret-manager://konf-test].
	// Here are other value(loader)s:
	//   - GCS(gs://konf-test/config.yaml)
	//   - Embedded FS(fs:///config/config.yaml)
}

func loadConfig(ctx context.Context) func() {
	config := konf.New(konf.WithOnStatus(func(loader konf.Loader, changed bool, err error) {
		fmt.Printf("load executed: loader=%v, changed=%v, error=%v\n", loader, changed, err)
	}))

	// Load configuration from embed file system.
	err := config.Load(fs.New(configFS, "config/config.yaml", fs.WithUnmarshal(yaml.Unmarshal)))
	if err != nil {
		panic(err) // handle error
	}
	// Load configuration from environment variables.
	err = config.Load(env.New())
	if err != nil {
		panic(err) // handle error
	}

	// Load configuration from GCP Cloud Storage.
	gcsLoader := gcs.New(
		"gs://konf-test/config.yaml",
		gcs.WithUnmarshal(yaml.Unmarshal),
		gcs.WithPollInterval(15*time.Second),
	)
	err = config.Load(gcsLoader)
	if err != nil {
		panic(err) // handle error
	}
	// Load configuration from GCP Secret Manager.
	secretManagerLoader := secretmanager.New(
		secretmanager.WithProject("konf-test"),
		secretmanager.WithPollInterval(20*time.Second),
	)
	err = config.Load(secretManagerLoader)
	if err != nil {
		panic(err) // handle error
	}
	konf.SetDefault(config)

	// Watch the changes of configuration.
	go func() {
		err := config.Watch(ctx)
		if err != nil {
			panic(err) // handle error
		}
	}()

	// Notify the changes of configuration.
	notifier := pubsub.NewNotifier("konf-test", pubsub.WithProject("konf-test"))
	notifier.Register(gcsLoader, secretManagerLoader)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		err := notifier.Start(ctx)
		if err != nil {
			panic(err) // handle error
		}
	}()

	return func() {
		waitGroup.Wait()
	}
}

//go:embed config
var configFS embed.FS
