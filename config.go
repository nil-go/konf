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

	providers providers
	onChanges onChanges
	watched   atomic.Bool
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
// This method is concurrent-safe.
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
	c.providers.append(loader, values)

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

	value := c.providers.sub(c.splitPath(path))
	if value == nil {
		return nil
	}

	converter := c.converter
	if converter == nil { // To support zero Config
		converter = defaultConverter
	}
	if err := converter.Convert(value, target); err != nil {
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

func (c *Config) splitPath(path string) []string {
	if path == "" {
		return nil
	}
	if !c.caseSensitive {
		path = defaultKeyMap(path)
	}

	return strings.Split(path, c.delim())
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

	value := c.providers.sub(c.splitPath(path))
	if value == nil {
		return path + " has no configuration.\n\n"
	}
	explanation := &strings.Builder{}
	c.explain(explanation, path, value)

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
	c.providers.traverse(func(provider *provider) {
		if v := maps.Sub(*provider.values.Load(), c.splitPath(path)); v != nil {
			loaders = append(loaders, loaderValue{provider.loader, v})
		}
	})
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

type (
	providers struct {
		providers []*provider
		values    atomic.Pointer[map[string]any]
		mutex     sync.RWMutex
	}
	provider struct {
		loader Loader
		values atomic.Pointer[map[string]any]
	}
)

func (p *providers) append(loader Loader, values map[string]any) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	provider := &provider{loader: loader}
	provider.values.Store(&values)
	p.providers = append(p.providers, provider)

	p.sync()
}

func (p *providers) changed() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.sync()
}

func (p *providers) sync() {
	values := make(map[string]any)
	for _, w := range p.providers {
		maps.Merge(values, *w.values.Load())
	}
	p.values.Store(&values)
}

func (p *providers) traverse(action func(*provider)) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, provider := range p.providers {
		action(provider)
	}
}

func (p *providers) sub(path []string) any {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	val := p.values.Load()
	if val == nil { // To support zero Config
		return nil
	}

	return maps.Sub(*val, path)
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
