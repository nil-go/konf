// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-viper/mapstructure/v2"

	"github.com/nil-go/konf/internal"
	"github.com/nil-go/konf/internal/credential"
	"github.com/nil-go/konf/internal/maps"
)

// Config reads configuration from appropriate sources.
//
// To create a new Config, call [New].
type Config struct {
	nocopy internal.NoCopy[Config]

	// Options.
	logger     *slog.Logger
	decodeHook mapstructure.DecodeHookFunc
	tagName    string
	delimiter  string

	// Loaded configuration.
	values    map[string]any
	providers []provider

	// For watching changes.
	onChanges      map[string][]func(*Config)
	onChangesMutex sync.RWMutex
	watched        atomic.Bool
}

// New creates a new Config with the given Option(s).
func New(opts ...Option) *Config {
	option := &options{}
	for _, opt := range opts {
		opt(option)
	}

	return (*Config)(option)
}

// Load loads configuration from the given loader.
// Each loader takes precedence over the loaders before it.
//
// This method can be called multiple times but it is not concurrency-safe.
func (c *Config) Load(loader Loader) error {
	c.nocopy.Check()

	if loader == nil {
		return errNilLoader
	}

	if c.values == nil {
		c.values = make(map[string]any)
	}

	values, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	maps.Merge(c.values, values)

	// Merged to empty map to convert to lower case.
	providerValues := make(map[string]any)
	maps.Merge(providerValues, values)
	c.providers = append(c.providers, provider{
		loader: loader,
		values: providerValues,
	})

	return nil
}

// Unmarshal reads configuration under the given path from the Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive.
func (c *Config) Unmarshal(path string, target any) error {
	if c == nil {
		return nil
	}

	c.nocopy.Check()

	decodeHook := c.decodeHook
	if decodeHook == nil {
		decodeHook = defaultDecodeHook
	}
	tagName := c.tagName
	if tagName == "" {
		tagName = "konf"
	}
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:           target,
			WeaklyTypedInput: true,
			DecodeHook:       decodeHook,
			TagName:          tagName,
		},
	)
	if err != nil {
		return fmt.Errorf("new decoder: %w", err)
	}

	if err := decoder.Decode(maps.Sub(c.values, c.split(path))); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

func (c *Config) split(key string) []string {
	delimiter := c.delimiter
	if delimiter == "" {
		delimiter = "."
	}

	return strings.Split(key, delimiter)
}

// Explain provides information about how Config resolve each value
// from loaders for the given path. It blur sensitive information.
// The path is case-insensitive.
func (c *Config) Explain(path string) string {
	if c == nil {
		return path + " has no configuration.\n\n"
	}

	c.nocopy.Check()

	explanation := &strings.Builder{}
	c.explain(explanation, path, maps.Sub(c.values, c.split(path)))

	return explanation.String()
}

func (c *Config) explain(explanation *strings.Builder, path string, value any) {
	delimiter := c.delimiter
	if delimiter == "" {
		delimiter = "."
	}

	if values, ok := value.(map[string]any); ok {
		for k, v := range values {
			c.explain(explanation, path+delimiter+k, v)
		}

		return
	}

	type loaderValue struct {
		loader Loader
		value  any
	}
	var loaders []loaderValue
	for _, provider := range c.providers {
		if v := maps.Sub(provider.values, c.split(path)); v != nil {
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
	explanation.WriteString(credential.Blur(path, loaders[0].value))
	explanation.WriteString("] that is loaded by loader[")
	explanation.WriteString(fmt.Sprintf("%v", loaders[0].loader))
	explanation.WriteString("].\n")
	if len(loaders) > 1 {
		explanation.WriteString("Here are other value(loader)s:\n")
		for _, loader := range loaders[1:] {
			explanation.WriteString("  - ")
			explanation.WriteString(credential.Blur(path, loader.value))
			explanation.WriteString("(")
			explanation.WriteString(fmt.Sprintf("%v", loader.loader))
			explanation.WriteString(")\n")
		}
	}
	explanation.WriteString("\n")
}

type provider struct {
	loader Loader
	values map[string]any
}

var (
	errNilLoader = errors.New("cannot load config from nil loader")

	defaultDecodeHook = mapstructure.ComposeDecodeHookFunc( //nolint:gochecknoglobals
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		mapstructure.TextUnmarshallerHookFunc(),
	)
)
