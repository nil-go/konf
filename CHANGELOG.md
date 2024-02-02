# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- add Config.Explain to provide information about how Config resolve each value from loaders (#78).
- add Default to get the default Config (#81).

### Changed

- Switch from mitchellh/mapstructure to go-viper/mapstructure (#69).

## [v0.3.0] - 11/17/2023

### Changed

- [BREAKING] Redesign API.

## [v0.2.0] - 3/18/2023

### Removed

- Remove file.WithLog to favor standard log.Printf (#32).

## [v0.1.0] - 3/12/2023

Initial alpha release.
