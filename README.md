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

Since factories can be reused, it's a good idea to define them in a specific package from which they can be imported. In
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

var PersonFactory = fabricator.New[types.Person](types.Person{}, fabricator.Options[types.Person]{
	Defaults: map[string]any{
		"FirstName": "Moishe",
		"LastName":  "Zuchmir",
		"Pets": func(iteration int, fieldName string) interface{} {
			pets := []types.Pet{}
			if iteration%2 == 0 {
				pets = append(pets, types.Pet{
					"Flippy",
					"Dolphin",
				})
			}
			return pets
		},
	},
})
```

As you can see above, the factory receives a `Defaults` object that maps struct field names, as map keys, to either
pre-specified values, or factory functions.

While factory functions are more verbose, they are a powerful way to generate data and they can of course be shared
across different fields or even different factories.

The signature for a factory function is `func(iteration int, fieldName string) interface{}`, with `iteration` being the
current value of the factory's internal counter, and `fieldName` being the name of the specific struct field for which a
value is being generated.

### Persistence Handler

When defining a factory, you can pass a `PersistenceHandler`, that is, a struct conforming to
the `fabricator.PersistenceHandler` interface:

```golang
package fabricator

type PersistenceHandler[T any] interface {
	Save(instance T) T
	SaveMany(instance []T) []T
}
```

With a persistence handler defined for the factory, you can call the `.Create` and `.CreateBatch` methods which build
and then persist the data in one command. For example:

```golang
package testhelpers

import (
	"github.com/Goldziher/fabricator"

	"github/someName/db"
	"github/someName/types"
)

type MyPersistenceHandler[T any] struct{}

func (handler MyPersistenceHandler[T]) Save(instance T) T {
	db.Create(&instance)
	return instance
}

func (handler MyPersistenceHandler[T]) SaveMany(instances []T) []T {
	db.Create(&instances)
	return instances
}

var PersonFactory = fabricator.New[types.Person](types.Person{}, fabricator.Options[types.Person]{
	PersistenceHandler: MyPersistenceHandler[types.Person]{},
})
```

## Factory Methods

Once a factory is defined it exposes the following methods:

### Build

`func (factory *Factory[T]) Build(overrides ...map[string]any) T`

Build creates a single instance of the factory's model:

```golang
package test_something

import (
	"testing"

	"github/someName/testhelpers"
)

func TestSomething(t *testing.T) {
	person := testhelpers.PersonFactory.Build()
	// ...
}
```

You can pass to build a mapping of override values, this works exactly like the factory defaults, for example:

```golang
package test_something

import (
	"testing"

	"github/someName/types"
	"github/someName/testhelpers"
)

func TestSomething(t *testing.T) {
	person := testhelpers.PersonFactory.Build(map[string]any{
		"FirstName": "Moishe",
		"LastName":  "Zuchmir",
		"Pets": func(iteration int, fieldName string) interface{} {
			pets := []types.Pet{}
			if iteration%2 == 0 {
				pets = append(pets, types.Pet{
					"Flippy",
					"Dolphin",
				})
			}
			return pets
		},
	})
	// ...
}
```

### Batch

`func (factory *Factory[T]) Batch(size int, overrides ...map[string]any) []T`

Batch builds a slice of instances of a given size:

```golang
package test_something

import (
	"testing"
	"github.com/stretchr/testify/assert"

	"github/someName/types"
	"github/someName/testhelpers"
)

func TestSomething(t *testing.T) {
	people := testhelpers.PersonFactory.Batch(5)
	assert.Len(t, people, 5)
}
```

Note: You can pass to batch overrides the same as you can for build

### Create

`func (factory *Factory[T]) Create(overrides ...map[string]any) T`

If a factory defines a [Persistence Handler](#persistence-handler) you can use `.Create` to build and persist a model
instance. Create is identical to `.Build` in terms of its API.

```golang
package test_something

import (
	"testing"

	"github/someName/testhelpers"
)

func TestSomething(t *testing.T) {
	person := testhelpers.PersonFactory.Create() // person is persisted using the PersistanceHandler's .Save method
	// ...
}
```

### CreateBatch

`func (factory *Factory[T]) CreateBatch(size int, overrides ...map[string]any) []T`

If a factory defines a [Persistence Handler](#persistence-handler) you can use `.CreateBatch` to build and persist a
slice of model instances of a given size. CreateBatch is identical to `.Batch` in terms of its API:

```golang
package test_something

import (
	"testing"

	"github/someName/testhelpers"
)

func TestSomething(t *testing.T) {
	people := testhelpers.PersonFactory.CreateBatch(5) // person is persisted using the PersistanceHandler's .SaveMany method
	// ...
}
```

## Using Struct Tags

Fabricator uses the excellent [faker](https://github.com/bxcodec/faker) library to generate mock data. As such, you can
use the faker struct tags to control the data generation, please consult the documentation for that library to see the
available tags.

## Contribution

This library is open to contributions. Please consult the [Contribution Guide](CONTRIBUTING.md).
