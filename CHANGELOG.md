# Changelog

All notable changes to Fabricator are documented in this file.

## v2.1.0 - 2026-09-02

Four additions aimed at the parts of a factory that were previously awkward: pinning generation so a failure reproduces, deriving a variant without restating a base, cycling values across a batch, and opting out of generation entirely.

### Added

- `Seed` for reproducible generation. It seeds both of faker's sources, including the separate one behind `faker:"uuid_*"` fields, and documents that it is process-wide, that it must not race a build, and that reproducibility depends on run scope.
- `Extend` for deriving a variant factory without restating a base's configuration.
- `Sequence` for cycling values by build iteration.
- `WithoutFaker` for starting builds from the zero value, and `WithFaker` to undo it on a derived factory.
- Benchmarks for `Build`, `Batch`, `Extend`, and the `WithoutFaker` path.
- A documentation site at <https://goldziher.github.io/fabricator/>, generated pixel-art brand assets, and a rewritten README.

### Changed

- Static `Value` and `Override` configuration is prepared once when the factory is configured rather than boxed on every build, and `Batch`/`CreateBatch` resolve their build options once per call rather than once per item. Together with skipping the per-build override config when a call has no overrides, this cuts allocations by 53% (geometric mean) across the build benchmarks: a `WithoutFaker` build goes from 4 allocations to 1, `Batch(10)` from 31 to 11, and `Batch(10, override)` from 41 to 13. `Extend` pays 232 to 304 bytes more per call in exchange, since preparation now happens at configuration time.
- Benchmarks were rewritten. The previous `BenchmarkBuild` was 99.6% `faker.FakeData` on a fixture carrying a slice and a map, where faker's default maximum collection size of 100 dominated everything; it read as a Fabricator cost and was not one. Benchmarks now separate the faker path from the `WithoutFaker` path, cover `Create`/`CreateBatch` and parallel builds, and hoist option construction out of `BenchmarkExtend`.
- Configuring the same field twice now keeps only the last configuration and does not run the superseded provider. Previously both ran and the second value won, so a superseded subfactory built a child and discarded it, and a superseded failing provider aborted a build whose value was never used.
- `Sequence` copies the values it is given rather than aliasing the caller's slice.
- The examples in `example_test.go` now assert their output, so they are executed rather than only compiled.
- Dropped the Go Report Card badge: the service is sunset and the badge now renders "go report: retired". A documentation badge takes its place, and the remaining badges use the project's cyan accent, matching the sibling projects.
- The website package now declares a description, license, and repository, and its lockfile no longer carries ten entries npm had marked extraneous.

### Fixed

- CI failed on every pull request: `poly fmt --check` rejected `poly.toml`, golangci-lint flagged trailing newlines in `example_test.go`, and `cargo install prek` no longer compiled. `prek` is removed, since the project uses poly and has no pre-commit configuration for it to run.

### Dependencies And Tooling

- Updated `github.com/go-faker/faker/v4` to v4.11.0, `github.com/stretchr/testify` to v1.12.1, and `golang.org/x/text` to v0.41.0.
- Updated golangci-lint to v2.13.2, the xberg-io reusable validate workflow to v1.11.4, and `github/codeql-action` to v4.37.9.
- Test coverage raised from 92.1% to 97.5%, with the new APIs verified by mutation testing.

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
