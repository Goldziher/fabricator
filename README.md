# Fabricator

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/Goldziher/fabricator)](https://goreportcard.com/report/github.com/Goldziher/fabricator)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=coverage)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=Goldziher_fabricator&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=Goldziher_fabricator)

</div>

Fabricator is a library for test data generation using structs and Go 1.18 generics. Its API is inspired by similar
libraries in other languages (e.g. [Pydantic-Factories](https://github.com/Goldziher/pydantic-factories)
, [Interface-Forge](https://github.com/Goldziher/interface-forge)), which was not possible in Go before the introduction
of generics.

## Installation

```shell
go get -u github.com/Goldziher/fabricator
```

## Example

```golang
package some_test

import (
	"testing"

	"github.com/Goldziher/fabricator"
	"github.com/stretchr/testify/assert"
)

type Pet struct {
	Name    string
	Species string
}

type Person struct {
	Id          int
	FirstName   string
	LastName    string
	Pets        []Pet
	FavoritePet Pet
}

var personFactory = fabricator.New[Person](Person{})

func TestSomething(t *testing.T) {
	personInstance := personFactory.Build()
	assert.IsType(t, Person{}, personInstance)
	assert.NotZero(t, personInstance.Id)
	assert.NotZero(t, personInstance.FirstName)
	assert.NotZero(t, personInstance.LastName)
	assert.NotZero(t, personInstance.Pets)
	assert.NotZero(t, personInstance.FavoritePet)
}
```

## Defining Factories

Defining a factory is very simple. Let's assume our app has a package called `types` where we define some struct, and
another package called `testhelpers` where we have some shared testing utilities.

```golang
package types

type Pet struct {
	Name    string
	Species string
}

type Person struct {
	Id          int
	FirstName   string
	LastName    string
	Pets        []Pet
	FavoritePet Pet
}
```

Since factories can be reused, its a good idea to define them in a specific package from which they can be imported. In
the case of our imaginary app, this will be the `testhelpers` package:

```golang
package testhelpers

import (
	"github.com/Goldziher/fabricator"

	"github/someName/types"
)

var PersonFactory = fabricator.New[types.Person](types.Person{})
```

Note: If we have use for a Pet without its nesting inside Person, we might also want to define a PetFactory, but this is
not required in this example, since a slice of pets will be generated inside the `PersonFactory`.

We could also pass an options object when defining the factory, setting the factory's `Defaults` and
a `PersistenceHandler` function.

### Factory Defaults

```golang
package testhelpers

import (
	"github.com/Goldziher/fabricator"

	"github/someName/types"
)

var PersonFactory = fabricator.New[types.Person](types.Person{}, fabricator.Options[Person]{
	Defaults: map[string]any{
		"FirstName": "Moishe",
		"LastName":  "Zuchmir",
		"Pets": []types.Pet{
			{
				"Flippy",
				"Dolphin",
			},
		},
	},
})
```

### Build

Build is the
