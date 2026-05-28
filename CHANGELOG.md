# Changelog

All notable changes to Fabricator are documented in this file.

## v2.0.0 - 2026-05-28

Fabricator v2 is a breaking generics-first redesign and uses the Go module path `github.com/Goldziher/fabricator/v2`.

### Breaking Changes

- Changed the module path from `github.com/Goldziher/fabricator` to `github.com/Goldziher/fabricator/v2`.
- Removed the `map[string]any` defaults and overrides API.
- Replaced string-first field configuration with typed `FieldRef[T, V]` descriptors from `FieldOf[T, V]`.
- Changed persistence handlers to accept `context.Context` and return errors.
- Changed `Create` and `CreateBatch` to require a context.
- Limited factories to non-pointer struct model types.

### Added

- Error-returning build and persistence APIs: `BuildE`, `BatchE`, `CreateE`, and `CreateBatchE`.
- Panic-on-error convenience wrappers remain for test ergonomics.
- `UnsafeFieldOf` as an explicit runtime-checked escape hatch.
- `AfterFaker`, `AfterBuild`, and `AfterCreate` lifecycle hooks.
- Context-aware subfactory helpers with parent-derived child overrides.
- Dynamic slice subfactory sizes.
- Explicit nil handling for nil-capable fields.
- Compile-tested examples in `example_test.go`.
- `CHANGELOG.md` with backfilled release history.
- `Taskfile.yml`, `prek`, `gitfluff`, `ai-rulez`, govulncheck, actionlint, and golangci-lint v2 maintenance workflow.

### Fixed

- Counter access is race-safe.
- Negative batch sizes return stable errors from `BatchE`.
- Nil subfactories fail immediately with clear panics.
- Field assignment no longer performs implicit type conversion.
- `AfterCreate` hooks receive the actual build iteration.
- Generated ai-rulez artifacts, including `.ai-rulez/.generated-manifest.json`, are ignored idempotently.
- Security reporting no longer directs users to public GitHub issues.

### Dependencies And Tooling

- Updated the project to Go 1.26.
- Updated `github.com/go-faker/faker/v4` to v4.7.0.
- Updated `github.com/stretchr/testify` to v1.11.1.
- Updated CI to current GitHub Actions majors, including `actions/checkout@v6`, `actions/upload-artifact@v7`, and CodeQL v4.

## v1.1.1 - 2024-02-16

### Fixed

- Fixed `CreateBatch` override handling so batch creation no longer takes only the first override.

### Dependencies And Tooling

- Updated Go dependencies.
- Updated GitHub Actions dependencies, including checkout, setup-go, CodeQL, upload/download artifact, cache, and golangci-lint action versions.
- Updated `gopkg.in/yaml.v3` to v3.0.0.

## v1.1.0 - 2023-08-08

### Changed

- Updated the project to Go 1.21.

### Dependencies And Tooling

- Added Dependabot.
- Updated pre-commit configuration.
- Updated GitHub Actions dependencies, including cache, checkout, setup-go, CodeQL, and golangci-lint action versions.

## v1.0.0 - 2022-04-22

### Added

- Initial public release of Fabricator.
- Added generic factory construction for Go structs.
- Added generated test data through `github.com/go-faker/faker/v4`.
- Added defaults and overrides using field-name maps.
- Added factory functions based on the factory counter and field name.
- Added batch building.
- Added persistence handlers with `Create` and `CreateBatch`.
- Added documentation and examples.
