# Fabricator

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/Goldziher/fabricator)](https://goreportcard.com/report/github.com/Goldziher/fabricator)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=coverage)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)

</div>

Fabricator is a generics-first Go library for building typed test data factories.

## Installation

```shell
go get github.com/Goldziher/fabricator
```

## Basic Usage

```go
package some_test

import (
	"testing"

	"github.com/Goldziher/fabricator"
	"github.com/stretchr/testify/require"
)

type Person struct {
	ID        int
	FirstName string
	LastName  string
}

func TestSomething(t *testing.T) {
	factory := fabricator.New(
		Person{},
		fabricator.Value[Person]("FirstName", "Moishe"),
		fabricator.Field[Person]("ID", func(ctx fabricator.BuildContext) int {
			return ctx.Iteration + 1
		}),
	)

	person := factory.Build(fabricator.Override[Person]("LastName", "Zuchmir"))

	require.Equal(t, 1, person.ID)
	require.Equal(t, "Moishe", person.FirstName)
	require.Equal(t, "Zuchmir", person.LastName)
}
```

## Typed Fields

Factory customization uses typed options:

```go
factory := fabricator.New(
	Person{},
	fabricator.Value[Person]("FirstName", "Moishe"),
	fabricator.Field[Person]("LastName", func(ctx fabricator.BuildContext) string {
		return "user-" + strconv.Itoa(ctx.Iteration)
	}),
)
```

`BuildContext` exposes the current `Iteration` and `FieldName`. Overrides use the same typed model:

```go
person := factory.Build(
	fabricator.Override[Person]("FirstName", "Chu"),
	fabricator.OverrideField[Person]("LastName", func(ctx fabricator.BuildContext) string {
		return fmt.Sprintf("Truong-%d", ctx.Iteration)
	}),
)
```

The old `map[string]any` defaults and overrides API has been removed. Invalid fields or values with incompatible types panic with explicit messages.

## Subfactories

Nested factories are first-class providers:

```go
type Pet struct {
	Name    string
	Species string
}

type Person struct {
	FavoritePet *Pet
	Pets        []Pet
}

petFactory := fabricator.New(
	Pet{},
	fabricator.Value[Pet]("Name", "Flippy"),
	fabricator.Value[Pet]("Species", "Dolphin"),
)

personFactory := fabricator.New(
	Person{},
	fabricator.Field[Person]("FavoritePet", fabricator.PtrSubfactory(petFactory)),
	fabricator.Field[Person]("Pets", fabricator.SliceSubfactory(petFactory, 2)),
)
```

Use `Subfactory` for value fields, `PtrSubfactory` for pointer fields, and `SliceSubfactory` for slices.

## Faker Options

Fabricator delegates base data generation to `github.com/go-faker/faker/v4`. Pass faker options through the factory:

```go
factory := fabricator.New(
	Person{},
	fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
)
```

## Persistence

`Create` and `CreateBatch` build data and pass it to a typed persistence handler:

```go
type PersistenceHandler[T any] interface {
	Save(instance T) T
	SaveMany(instances []T) []T
}

factory := fabricator.New(
	Person{},
	fabricator.WithPersistenceHandler[Person](handler),
)

person := factory.Create()
people := factory.CreateBatch(5)
```

## Development

This repository uses Go 1.26, golangci-lint v2, `prek`, `gitfluff`, and `ai-rulez`.

```shell
task setup
task test
task test:race
task check
task lint
```
