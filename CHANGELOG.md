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

## [0.3.0] - 2023-11-17

### Changed

- [BREAKING] Redesign API.

## [0.2.0] - 2023-03-18

### Removed

- Remove file.WithLog to favor standard log.Printf (#32).

## [0.1.0] - 2023-03-12

Initial alpha release.
