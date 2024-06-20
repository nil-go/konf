# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Overwriting parent context in `*Config.Watch()` what led to unwanted routine exit (#370).
- Use atomic.Pointer for Config.values and provider.values to avoid race condition (#378).

## [1.2.0] - 2024-06-10

### Changed

- [Breaking] The map key is case insensitive now. If you would like to keep it case sensitive.
  please add konf.WithMapKeyCaseSensitive option (#365).

## [1.1.1] - 2024-05-02

### Fixed

- Explain supports empty string as path (#314).
- Reserve the case for key of map when unmarshalling. All keys in map used to be lower case,
  now it matches the case in the configuration (#318).

## [1.1.0] - 2024-04-24

This version introduces a new feature to support change notification
via AWS SNS, GCP PubSub, and Azure Service Bus.

### Added

- Support change notification via SNS topic (#267).
- Support change notification via PubSub topic (#294).
- Support change notification via Service Bus topic (#302).
- Add provider for AWS Parameter Store (#298).

## [1.0.0] - 2024-03-16

First stable release.

## [0.9.2] - 2024-03-10

### Fixed

- Return no chang if s3/azblob/gcs returns 304 (not modified) (#233).

## [0.9.1] - 2024-03-10

### Fixed

- flag and pflag always add the default value even the key already exists
  since konf.Exists uses empty delimiter for empty Config (#228).

## [0.9.0] - 2024-03-10

### Added

- Add konf.WithCaseSensitive to support case-sensitive path match (#205).
- Add GCP Cloud Storage Loader (#210).
- Add AWS S3 Loader (#214).
- Add Azure Blob Storage Loader (#217).

## [0.8.1] - 2024-03-06

### Fixed

- Config uses default tag name of decode hooks even only one of them is set (#204).

## [0.8.0] - 2024-03-06

### Added

- Statuser interface for providers report status of configuration loading/watching (#199).

### Changed

- [Breaking] Replace mapstructure with simpler built-in converter (#198).

### Removed

- [Breaking] Remove WithLogHandler in providers in favor of Statuser interface (#199).

## [0.7.0] - 2024-02-29

### Changed

- [Breaking] Use pointer receiver for konf.Config to make empty Config useful (#187).

### Removed

- [Breaking] Remove konf.Default() to disallow loading configuration into the default Config (#180).
- [Breaking] Remove ExplainOption from Config.Explain for always blurring sensitive information (#180).
- [Breaking] Remove LoadOption from Config.Load (#184).

## [0.6.3] - 2024-02-23

### Fixed

- The changed values in watch may not update values in config (#171).

## [0.6.2] - 2024-02-21

### Fixed

- Add ContinueOnError so watcher can continue watching even the loader fails to load the configuration (#161).

## [0.6.1] - 2024-02-21

### Changed

- merge loader into providers even it fails the loading. Developers can ignore the loading error
  and wait for the watching to get latest configuration (#159).

## [0.6.0] - 2024-02-19

### Changed

- [BREAKING] Change signature of valueFormatter for config.Explain (#146).

### Removed

- Remove deprecated env.WithDelimiter/flag.WithDelimiter/pflag.WithDelimiter in favor of WithNameSplitter (#137).

## [0.5.0] - 2024-02-16

### Added

- Add env.WithNameSplitter/flag.WithNameSplitter/pflag.WithNameSplitter to split the name of the flag/env (#110).
- Add Azure App Configuration Loader (#121).
- Add GCP Secret Manager Loader (#128).

### Changed

- Use CredentialFormatter to blur sensitive information in config.Explain (#113).

### Deprecated

- Deprecate env.WithDelimiter/flag.WithDelimiter/pflag.WithDelimiter in favor of WithNameSplitter (#110).

## [0.4.0] - 2024-02-07

### Added

- add Config.Explain to provide information about how Config resolve each value from loaders (#78).
- add Default to get the default Config (#81).
- add AWS AppConfig Loader (#92).

### Changed

- Switch from mitchellh/mapstructure to go-viper/mapstructure (#69).

## [0.3.0] - 2023-11-17

### Changed

- [BREAKING] Redesign API.

## [0.2.0] - 2023-03-18

### Removed

- Remove file.WithLog to favor standard log.Printf (#32).

## [0.1.0] - 2023-03-12

Initial alpha release.
