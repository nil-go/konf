# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
