// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package konf

import (
	"encoding"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

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
	caseSensitive bool
	delimiter     string
	logger        *slog.Logger
	onStatus      func(loader Loader, changed bool, err error)
	converter     convert.Converter

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

	if len(option.hooks) == 0 {
		option.hooks = defaultHooks
	}
	if option.tagName == "" {
		option.hooks = append(option.hooks, defaultTagName)
	} else {
		option.hooks = append(option.hooks, convert.WithTagName(option.tagName))
	}
	if !option.caseSensitive {
		option.hooks = append(option.hooks, defaultKeyMap)
	}
	option.converter = convert.New(option.hooks...)

	return &(option.Config)
}

// Load loads configuration from the given loader.
// Each loader takes precedence over the loaders before it.
//
// This method can be called multiple times but it is not concurrency-safe.
func (c *Config) Load(loader Loader) error {
	c.nocopy.Check()

	if loader == nil {
		return nil
	}

	if c.values == nil {
		c.values = make(map[string]any)
	}

	if statuser, ok := loader.(Statuser); ok {
		statuser.Status(func(changed bool, err error) {
			if err != nil {
				c.logger.Warn(
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

	values, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	prd := provider{
		loader: loader,
		values: c.transformKeys(values),
	}
	c.providers = append(c.providers, prd)
	maps.Merge(c.values, prd.values)

	return nil
}

// Unmarshal reads configuration under the given path from the Config
// and decodes it into the given object pointed to by target.
// The path is case-insensitive unless konf.WithCaseSensitive is set.
func (c *Config) Unmarshal(path string, target any) error {
	if c == nil {
		return nil
	}

	c.nocopy.Check()

	converter := c.converter
	if reflect.ValueOf(converter).IsZero() {
		converter = defaultConverter
	}

	if err := converter.Convert(c.sub(c.values, path), target); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}

func (c *Config) sub(values map[string]any, path string) any {
	delimiter := c.delimiter
	if delimiter == "" {
		delimiter = "."
	}
	if !c.caseSensitive {
		path = toLower(path)
	}

	return maps.Sub(values, strings.Split(path, delimiter))
}

func (c *Config) transformKeys(m map[string]any) map[string]any {
	if c.caseSensitive {
		return m
	}

	return maps.TransformKeys(m, toLower)
}

func toLower(s string) string {
	return strings.Map(unicode.ToLower, s)
}

// Explain provides information about how Config resolve each value
// from loaders for the given path. It blur sensitive information.
// The path is case-insensitive unless konf.WithCaseSensitive is set.
func (c *Config) Explain(path string) string {
	if c == nil {
		return path + " has no configuration.\n\n"
	}

	c.nocopy.Check()

	explanation := &strings.Builder{}
	c.explain(explanation, path, c.sub(c.values, path))

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
		if v := c.sub(provider.values, path); v != nil {
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

//nolint:gochecknoglobals
var (
	defaultTagName = convert.WithTagName("konf")
	defaultKeyMap  = convert.WithKeyMapper(toLower)
	defaultHooks   = []convert.Option{
		convert.WithHook[string, time.Duration](time.ParseDuration),
		convert.WithHook[string, []string](func(f string) ([]string, error) {
			return strings.Split(f, ","), nil
		}),
		convert.WithHook[string, encoding.TextUnmarshaler](func(f string, t encoding.TextUnmarshaler) error {
			return t.UnmarshalText([]byte(f)) //nolint:wrapcheck
		}),
	}
	defaultConverter = convert.New(
		append(defaultHooks, defaultTagName, defaultKeyMap)...,
	)
)
