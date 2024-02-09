// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

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

	values    map[string]any
	providers []*provider

	onChanges      map[string][]func(*Config)
	onChangesMutex sync.RWMutex
	watchOnce      sync.Once
}

type provider struct {
	loader Loader
	values map[string]any
}

// New creates a new Config with the given Option(s).
func New(opts ...Option) *Config {
	option := &options{
		values:    make(map[string]any),
		onChanges: make(map[string][]func(*Config)),
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

	return (*Config)(option)
}

// Load loads configuration from the given loader.
// Each loader takes precedence over the loaders before it.
//
// This method can be called multiple times but it is not concurrency-safe.
// It panics if loader is nil.
func (c *Config) Load(loader Loader) error {
	if loader == nil {
		panic("cannot load config from nil loader")
	}

	values, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	maps.Merge(c.values, values)

	provider := &provider{
		loader: loader,
		values: make(map[string]any),
	}
	// Merged to empty map to convert to lower case.
	maps.Merge(provider.values, values)
	c.providers = append(c.providers, provider)

	return nil
}

// Unmarshal reads configuration under the given path from the Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive.
func (c *Config) Unmarshal(path string, target any) error {
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

	if err := decoder.Decode(sub(c.values, strings.Split(path, c.delimiter))); err != nil {
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
func (c *Config) Explain(path string, opts ...ExplainOption) string {
	option := &explainOptions{}
	for _, opt := range opts {
		opt(option)
	}
	if option.valueFormatter == nil {
		option.valueFormatter = func(_ string, _ Loader, value any) string {
			return fmt.Sprint(value)
		}
	}

	explanation := &strings.Builder{}
	c.explain(explanation, path, sub(c.values, strings.Split(path, c.delimiter)), *option)

	return explanation.String()
}

func (c *Config) explain(explanation *strings.Builder, path string, value any, option explainOptions) {
	if values, ok := value.(map[string]any); ok {
		for k, v := range values {
			c.explain(explanation, path+c.delimiter+k, v, option)
		}

		return
	}

	var loaders []loaderValue
	for _, provider := range c.providers {
		if v := sub(provider.values, strings.Split(path, c.delimiter)); v != nil {
			loaders = append(loaders, loaderValue{provider.loader, v})
		}
	}
	slices.Reverse(loaders)

	if len(loaders) == 0 {
		_, _ = fmt.Fprintf(explanation, "%s has no configuration.\n\n", path)

		return
	}
	_, _ = fmt.Fprintf(explanation, "%s has value[%s] that is loaded by loader[%v].\n",
		path, option.valueFormatter(path, loaders[0].loader, loaders[0].value), loaders[0].loader,
	)
	if len(loaders) > 1 {
		_, _ = fmt.Fprintf(explanation, "Here are other value(loader)s:\n")
		for _, loader := range loaders[1:] {
			_, _ = fmt.Fprintf(explanation, "  - %s(%v)\n",
				option.valueFormatter(path, loader.loader, loader.value), loader.loader,
			)
		}
	}
	explanation.WriteString("\n")
}

type loaderValue struct {
	loader Loader
	value  any
}
