# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.0.4] - 2026-11-24

### Added

- Show APIM subscription owner names in the list command.
- Add `--show-keys` to display subscription keys explicitly.

### Changed

- Hide subscription keys in list output by default.
- Update Azure SDK dependencies and GitHub Actions.

## [0.0.3] - 2025-01-01

### Changed

- Use git tags to manage releases
- Run tests only on app changes

### Fixed

- Add warning for skipping the master key

## [0.0.2] - 2024-12-01

### Fixed

- Master key handling
- Restore APIs with global scope

## [0.0.1] - 2024-11-01

### Added

- Initial release
- Backup and restore subscription keys from Azure API Management
- List, compare, clean, and delete commands
