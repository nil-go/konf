// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"context"
	"encoding"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nil-go/konf/internal"
	"github.com/nil-go/konf/internal/convert"
	"github.com/nil-go/konf/internal/credential"
	"github.com/nil-go/konf/internal/maps"
)

// Config reads configuration from appropriate sources.
//
// To create a new Config, call [New].
type Config struct {
	nocopy internal.NoCopy[Config]

	// Options.
	caseSensitive       bool
	mapKeyCaseSensitive bool
	delimiter           string
	logger              *slog.Logger
	onStatus            func(loader Loader, changed bool, err error)
	converter           *convert.Converter

	// Loaded configuration.
	values         atomic.Pointer[map[string]any]
	providers      []*provider
	providersMutex sync.RWMutex

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

	// Build converter from options.
	if len(option.convertOpts) == 0 {
		option.convertOpts = defaultHooks
	}
	if option.tagName == "" {
		option.tagName = defaultTagName
	}
	option.convertOpts = append(option.convertOpts, convert.WithTagName(option.tagName))
	if !option.caseSensitive {
		option.convertOpts = append(option.convertOpts, convert.WithKeyMapper(defaultKeyMap))
	}
	option.converter = convert.New(option.convertOpts...)

	return &(option.Config)
}

// Load loads configuration from the given loader.
// Each loader takes precedence over the loaders before it.
//
// This method can be called multiple times but it is not concurrency-safe.
func (c *Config) Load(loader Loader) error {
	if loader == nil {
		return nil
	}
	c.nocopy.Check()

	// Load values into a new provider.
	values, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	c.transformKeys(values)
	prd := provider{
		loader: loader,
	}
	prd.values.Store(&values)
	c.providersMutex.Lock()
	c.providers = append(c.providers, &prd)
	c.providersMutex.Unlock()

	// Merge loaded values into values map.
	if c.values.Load() == nil {
		c.values.Store(&map[string]any{})
	}
	maps.Merge(*c.values.Load(), *prd.values.Load())

	if _, ok := loader.(Watcher); !ok {
		return nil
	}

	// Special handling if loader is also watcher.
	if c.watched.Load() {
		c.log(context.Background(),
			slog.LevelWarn,
			"The Watch on loader has no effect as Config.Watch has been executed.",
			slog.Any("loader", loader),
		)

		return nil
	}

	if statuser, ok := loader.(Statuser); ok {
		statuser.Status(func(changed bool, err error) {
			if err != nil {
				c.log(context.Background(),
					slog.LevelWarn,
					"Error when loading configuration.",
					slog.Any("loader", loader),
					slog.Any("error", err),
				)
			}
			if c.onStatus != nil {
				c.onStatus(loader, changed, err)
			}
		})
	}

	return nil
}

// Unmarshal reads configuration under the given path from the Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive unless konf.WithCaseSensitive is set.
func (c *Config) Unmarshal(path string, target any) error {
	if c == nil { // To support nil
		return nil
	}
	c.nocopy.Check()

	if c.values.Load() == nil {
		return nil // To support zero Config
	}

	converter := c.converter
	if converter == nil { // To support zero Config
		converter = defaultConverter
	}
	if err := converter.Convert(c.sub(*c.values.Load(), path), target); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

func (c *Config) log(ctx context.Context, level slog.Level, message string, attrs ...slog.Attr) {
	logger := c.logger
	if c.logger == nil { // To support zero Config
		logger = slog.Default()
	}
	logger.LogAttrs(ctx, level, message, attrs...)
}

func (c *Config) sub(values map[string]any, path string) any {
	if !c.caseSensitive {
		path = defaultKeyMap(path)
	}

	return maps.Sub(values, path, c.delim())
}

func (c *Config) delim() string {
	if c.delimiter == "" { // To support zero Config
		return "."
	}

	return c.delimiter
}

func (c *Config) transformKeys(m map[string]any) {
	if !c.caseSensitive {
		maps.TransformKeys(m, defaultKeyMap, c.mapKeyCaseSensitive)
	}
}

// Explain provides information about how Config resolve each value
// from loaders for the given path. It blur sensitive information.
// The path is case-insensitive unless konf.WithCaseSensitive is set.
func (c *Config) Explain(path string) string {
	if c == nil { // To support nil
		return path + " has no configuration.\n\n"
	}
	c.nocopy.Check()

	if c.values.Load() == nil { // To support zero Config
		return path + " has no configuration.\n\n"
	}

	explanation := &strings.Builder{}
	c.explain(explanation, path, c.sub(*c.values.Load(), path))

	return explanation.String()
}

func (c *Config) explain(explanation *strings.Builder, path string, value any) {
	if values, ok := value.(map[string]any); ok {
		for key, val := range values {
			newPath := path
			if newPath != "" {
				newPath += c.delim()
			}
			newPath += key
			c.explain(explanation, newPath, val)
		}

		return
	}

	type loaderValue struct {
		loader Loader
		value  any
	}
	var loaders []loaderValue
	for _, provider := range c.providers {
		if v := c.sub(*provider.values.Load(), path); v != nil {
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
	values atomic.Pointer[map[string]any]
}

//nolint:gochecknoglobals
var (
	defaultTagName = "konf"
	defaultKeyMap  = strings.ToLower

	defaultHooks = []convert.Option{
		convert.WithHook[string, time.Duration](time.ParseDuration),
		convert.WithHook[string, []string](func(f string) ([]string, error) {
			return strings.Split(f, ","), nil
		}),
		convert.WithHook[string, encoding.TextUnmarshaler](func(f string, t encoding.TextUnmarshaler) error {
			return t.UnmarshalText(internal.String2ByteSlice(f))
		}),
	}
	defaultConverter = convert.New(
		append(defaultHooks, convert.WithTagName(defaultTagName), convert.WithKeyMapper(defaultKeyMap))...,
	)
)
