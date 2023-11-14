// Copyright (c) 2023 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

// Package konf defines a general-purpose configuration API and abstract interfaces
// to back that API. Packages in the Go ecosystem can depend on this package,
// while callers can load configuration from whatever source is appropriate.
//
// # Usage
//
// Reading configuration is done using a Config instance. Config is a concrete type
// with methods, which loads the actual configuration from a Loader interface,
// and reloads latest configuration when it has changes from a Watcher interface.
//
// Config has following main methods:
//   - Config.Watch reloads configuration when it changes.
//   - Config.Unmarshal loads configuration under the given path
//     into the given object pointed to by target.
//   - Config.OnChange register callback on configuration changes.
//
// # Global Config
//
// The following package's functions load configuration
// from the global Config while it is set by SetGlobal:
//
//   - Get instances the given type and loads configuration into it.
//     It returns zero value if there is an error while getting configuration.
//   - Unmarshal loads configuration under the given path
//     into the given object pointed to by target.
//   - OnChange register callback on configuration changes.
package konf
