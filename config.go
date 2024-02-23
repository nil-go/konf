// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-viper/mapstructure/v2"

	"github.com/nil-go/konf/internal/maps"
)

// Config reads configuration from appropriate sources.
//
// To create a new Config, call [New].
type Config struct {
	logger     *slog.Logger
	decodeHook mapstructure.DecodeHookFunc
	tagName    string
	delimiter  string

	values *values
}

type (
	provider struct {
		loader Loader
		values map[string]any
	}

	values struct {
		values    map[string]any
		providers []provider

		onChanges      map[string][]func(Config)
		onChangesMutex sync.RWMutex
		watched        atomic.Bool
	}
)

type DecodeHook any

// New creates a new Config with the given Option(s).
func New(opts ...Option) Config {
	option := &options{
		values: &values{
			values:    make(map[string]any),
			onChanges: make(map[string][]func(Config)),
		},
	}
	for _, opt := range opts {
		opt(option)
	}
	if option.logger == nil {
		option.logger = slog.Default()
	}
	if option.delimiter == "" {
		option.delimiter = "."
	}
	if option.tagName == "" {
		option.tagName = "konf"
	}
	if option.decodeHook == nil {
		option.decodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.TextUnmarshallerHookFunc(),
		)
	}

	return Config(*option)
}

// Load loads configuration from the given loader.
// Each loader takes precedence over the loaders before it.
//
// This method can be called multiple times but it is not concurrency-safe.
// It panics if loader is nil.
func (c Config) Load(loader Loader, opts ...LoadOption) error {
	if loader == nil {
		panic("cannot load config from nil loader")
	}

	loadOption := &loadOptions{}
	for _, opt := range opts {
		opt(loadOption)
	}

	values, err := loader.Load()
	if err != nil {
		if !loadOption.continueOnError {
			return fmt.Errorf("load configuration: %w", err)
		}
		c.logger.LogAttrs(
			context.Background(), slog.LevelWarn,
			"failed to load configuration",
			slog.Any("loader", loader),
			slog.Any("error", err),
		)
	}
	maps.Merge(c.values.values, values)

	// Merged to empty map to convert to lower case.
	providerValues := make(map[string]any)
	maps.Merge(providerValues, values)
	c.values.providers = append(c.values.providers, provider{
		loader: loader,
		values: providerValues,
	})

	return nil
}

// Unmarshal reads configuration under the given path from the Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive.
func (c Config) Unmarshal(path string, target any) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:           target,
			WeaklyTypedInput: true,
			DecodeHook:       c.decodeHook,
			TagName:          c.tagName,
		},
	)
	if err != nil {
		return fmt.Errorf("new decoder: %w", err)
	}

	if err := decoder.Decode(sub(c.values.values, strings.Split(path, c.delimiter))); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

func sub(values map[string]any, keys []string) any {
	if len(keys) == 0 || len(keys) == 1 && keys[0] == "" {
		return values
	}

	var next any = values
	for _, key := range keys {
		key = strings.ToLower(key)
		mp, ok := next.(map[string]any)
		if !ok {
			return nil
		}

		val, exist := mp[key]
		if !exist {
			return nil
		}
		next = val
	}

	return next
}

// Explain provides information about how Config resolve each value
// from loaders for the given path.
// The path is case-insensitive.
//
// If there are sensitive information (e.g. password, secret) which should not be exposed,
// you can use [WithValueFormatter] to pass a value formatter to blur the information.
// By default, it uses CredentialFormatter to blur sensitive information.
func (c Config) Explain(path string, opts ...ExplainOption) string {
	option := &explainOptions{}
	for _, opt := range opts {
		opt(option)
	}
	if option.valueFormatter == nil {
		option.valueFormatter = CredentialFormatter
	}

	explanation := &strings.Builder{}
	c.explain(explanation, path, sub(c.values.values, strings.Split(path, c.delimiter)), *option)

	return explanation.String()
}

func (c Config) explain(explanation *strings.Builder, path string, value any, option explainOptions) {
	if values, ok := value.(map[string]any); ok {
		for k, v := range values {
			c.explain(explanation, path+c.delimiter+k, v, option)
		}

		return
	}

	var loaders []loaderValue
	for _, provider := range c.values.providers {
		if v := sub(provider.values, strings.Split(path, c.delimiter)); v != nil {
			loaders = append(loaders, loaderValue{provider.loader, v})
		}
	}
	slices.Reverse(loaders)

	if len(loaders) == 0 {
		explanation.WriteString(path)
		explanation.WriteString(" has no configuration.\n\n")

		return
	}
	explanation.WriteString(path)
	explanation.WriteString(" has value[")
	explanation.WriteString(option.valueFormatter(loaders[0].loader, path, loaders[0].value))
	explanation.WriteString("] that is loaded by loader[")
	explanation.WriteString(fmt.Sprintf("%v", loaders[0].loader))
	explanation.WriteString("].\n")
	if len(loaders) > 1 {
		explanation.WriteString("Here are other value(loader)s:\n")
		for _, loader := range loaders[1:] {
			explanation.WriteString("  - ")
			explanation.WriteString(option.valueFormatter(loader.loader, path, loader.value))
			explanation.WriteString("(")
			explanation.WriteString(fmt.Sprintf("%v", loader.loader))
			explanation.WriteString(")\n")
		}
	}
	explanation.WriteString("\n")
}

type loaderValue struct {
	loader Loader
	value  any
}
