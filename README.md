# Fabricator

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/Goldziher/fabricator)](https://goreportcard.com/report/github.com/Goldziher/fabricator)
[![CI](https://github.com/Goldziher/fabricator/actions/workflows/ci.yaml/badge.svg)](https://github.com/Goldziher/fabricator/actions/workflows/ci.yaml)
[![Latest Release](https://img.shields.io/github/v/release/Goldziher/fabricator?sort=semver)](https://github.com/Goldziher/fabricator/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Goldziher/fabricator)](https://github.com/Goldziher/fabricator/blob/main/go.mod)
[![License](https://img.shields.io/github/license/Goldziher/fabricator)](https://github.com/Goldziher/fabricator/blob/main/LICENSE)

</div>

Fabricator is a generics-first Go library for building typed test data factories.

## Installation

```shell
go get github.com/Goldziher/fabricator/v2
```

## Basic Usage

```go
type Person struct {
	ID        int
	FirstName string
	LastName  string
}

firstName := fabricator.FieldOf[Person, string]("FirstName")
lastName := fabricator.FieldOf[Person, string]("LastName")
id := fabricator.FieldOf[Person, int]("ID")
factory := fabricator.New(
	Person{},
	fabricator.Value(firstName, "Moishe"),
	fabricator.Field(id, func(ctx fabricator.BuildContext) int {
		return ctx.Iteration + 1
	}),
)

person := factory.Build(fabricator.Override(lastName, "Zuchmir"))
```

See `example_test.go` for compile-tested examples.

## Typed Fields

Factory customization uses typed field descriptors:

```go
firstName := fabricator.FieldOf[Person, string]("FirstName")
factory := fabricator.New(
	Person{},
	fabricator.Value(firstName, "Moishe"),
)
```

`FieldOf[T, V]` validates that the field exists, is exported, and accepts `V`. `UnsafeFieldOf` is available as an explicit runtime-checked escape hatch.

The old `map[string]any` defaults and overrides API has been removed in v2.

## Errors And Panics

Every build and persistence method has an error-returning form:

```go
person, err := factory.BuildE()
people, err := factory.BatchE(5)
created, err := factory.CreateE(ctx)
createdBatch, err := factory.CreateBatchE(ctx, 5)
```

`Build`, `Batch`, `Create`, and `CreateBatch` remain panic-on-error convenience wrappers for terse tests.

## Lifecycle Hooks

Hooks can inspect or mutate the generated object:

```go
factory := fabricator.New(
	Person{},
	fabricator.AfterBuild(func(person *Person, ctx fabricator.BuildContext) error {
		person.Email = strings.ToLower(person.FirstName) + "@example.com"
		return nil
	}),
)
```

`AfterFaker` runs after faker data generation, `AfterBuild` runs after defaults and overrides, and `AfterCreate` runs after persistence.

## Subfactories

Nested factories are first-class providers:

```go
petFactory := fabricator.New(Pet{}, fabricator.Value(fabricator.FieldOf[Pet, string]("Name"), "Flippy"))
personFactory := fabricator.New(
	Person{},
	fabricator.Field(fabricator.FieldOf[Person, *Pet]("FavoritePet"), fabricator.PtrSubfactory(petFactory)),
	fabricator.Field(fabricator.FieldOf[Person, []Pet]("Pets"), fabricator.SliceSubfactory(petFactory, 2)),
)
```

Use `Subfactory` for value fields, `PtrSubfactory` for pointer fields, and `SliceSubfactory` for slices. The `*With` variants let child overrides and slice sizes depend on the parent `BuildContext`.

## Faker Options

Fabricator delegates base data generation to `github.com/go-faker/faker/v4`:

```go
factory := fabricator.New(
	Person{},
	fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
)
```

## Persistence

Persistence handlers are context-aware and return errors:

```go
type PersistenceHandler[T any] interface {
	Save(ctx context.Context, instance T) (T, error)
	SaveMany(ctx context.Context, instances []T) ([]T, error)
}
```

## Counter Semantics

The counter is race-safe. Iterations are consumed after faker succeeds and before defaults, overrides, and hooks run. `SetCounter` and `ResetCounter` are race-safe but are not coordination primitives for concurrent builds.

## Limits

Factories are for non-pointer struct types. Nested dotted field paths are not supported; use hooks for nested mutation.

## Development

This repository uses Go 1.26, golangci-lint v2, `prek`, `gitfluff`, and `ai-rulez`.

```shell
task setup
task test
task test:race
task check
task lint
```
