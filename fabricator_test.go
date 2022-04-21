package fabricator_test

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

func TestNew(t *testing.T) {
	t.Run("Success Scenario", func(t *testing.T) {
		assert.NotPanics(t, func() { _ = fabricator.New(Person{}) })
	})
	t.Run("Failure Scenario", func(t *testing.T) {
		assert.Panics(t, func() { _ = fabricator.New(100) })
	})
}

func TestFactory_Build(t *testing.T) {
	t.Run("Test .Build", func(t *testing.T) {
		factory := fabricator.New(Person{})
		person := factory.Build()
		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.Id)
		assert.NotZero(t, person.FirstName)
		assert.NotZero(t, person.LastName)
		assert.NotZero(t, person.Pets)
		assert.NotZero(t, person.FavoritePet)
	})
	t.Run("Test .Build defaults", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.Options[Person]{
			Defaults: map[string]any{
				"FirstName": "Moishe",
				"LastName":  "Zuchmir",
				"Pets": []Pet{
					{
						"Flippy",
						"Dolphin",
					},
				},
			},
		})
		person := factory.Build()
		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.Id)
		assert.NotZero(t, person.FavoritePet)
		assert.Len(t, person.Pets, 1)

		assert.Equal(t, person.FirstName, "Moishe")
		assert.Equal(t, person.LastName, "Zuchmir")

		pet := person.Pets[0]
		assert.Equal(t, pet.Name, "Flippy")
		assert.Equal(t, pet.Species, "Dolphin")
	})
	t.Run("Test .Build overrides", func(t *testing.T) {
		factory := fabricator.New(Person{})
		person := factory.Build(map[string]interface{}{
			"FirstName": "Moishe",
			"LastName":  "Zuchmir",
			"Pets": []Pet{
				{
					"Flippy",
					"Dolphin",
				},
			},
		})
		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.Id)
		assert.NotZero(t, person.FavoritePet)
		assert.Len(t, person.Pets, 1)

		assert.Equal(t, person.FirstName, "Moishe")
		assert.Equal(t, person.LastName, "Zuchmir")

		pet := person.Pets[0]
		assert.Equal(t, pet.Name, "Flippy")
		assert.Equal(t, pet.Species, "Dolphin")
	})
	t.Run("Panic Scenario", func(t *testing.T) {
		ttype := struct {
			Key interface{}
		}{}
		factory := fabricator.New(ttype)
		assert.Panics(t, func() {
			_ = factory.Build()
		})
	})
}

func TestFactory_Batch(t *testing.T) {
	t.Run("Test .Batch", func(t *testing.T) {
		factory := fabricator.New(Person{})
		people := factory.Batch(5)
		assert.Len(t, people, 5)
		for _, person := range people {
			assert.IsType(t, Person{}, person)
			assert.NotZero(t, person.Id)
			assert.NotZero(t, person.FirstName)
			assert.NotZero(t, person.LastName)
			assert.NotZero(t, person.Pets)
			assert.NotZero(t, person.FavoritePet)
		}
	})
	t.Run("Test .Batch defaults", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.Options[Person]{
			Defaults: map[string]any{
				"FirstName": "Moishe",
				"LastName":  "Zuchmir",
				"Pets": []Pet{
					{
						"Flippy",
						"Dolphin",
					},
				},
			},
		})
		people := factory.Batch(5)
		assert.Len(t, people, 5)
		for _, person := range people {
			assert.IsType(t, Person{}, person)
			assert.NotZero(t, person.Id)
			assert.NotZero(t, person.FavoritePet)
			assert.Len(t, person.Pets, 1)

			assert.Equal(t, person.FirstName, "Moishe")
			assert.Equal(t, person.LastName, "Zuchmir")

			pet := person.Pets[0]
			assert.Equal(t, pet.Name, "Flippy")
			assert.Equal(t, pet.Species, "Dolphin")
		}
	})
	t.Run("Test .Build overrides", func(t *testing.T) {
		factory := fabricator.New(Person{})
		people := factory.Batch(5, map[string]interface{}{
			"FirstName": "Moishe",
			"LastName":  "Zuchmir",
			"Pets": []Pet{
				{
					"Flippy",
					"Dolphin",
				},
			},
		})
		assert.Len(t, people, 5)
		for _, person := range people {
			assert.IsType(t, Person{}, person)
			assert.NotZero(t, person.Id)
			assert.NotZero(t, person.FavoritePet)
			assert.Len(t, person.Pets, 1)

			assert.Equal(t, person.FirstName, "Moishe")
			assert.Equal(t, person.LastName, "Zuchmir")

			pet := person.Pets[0]
			assert.Equal(t, pet.Name, "Flippy")
			assert.Equal(t, pet.Species, "Dolphin")
		}
	})
}

func TestFactory_Create(t *testing.T) {
	t.Run("Success Scenario", func(t *testing.T) {
		var result Person

		factory := fabricator.New[Person](Person{}, fabricator.Options[Person]{
			PersistenceHandler: func(instance Person) Person {
				result = instance
				return instance
			},
		})
		person := factory.Create()

		assert.NotNil(t, result)
		assert.Equal(t, person, result)
		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.Id)
		assert.NotZero(t, person.FirstName)
		assert.NotZero(t, person.LastName)
		assert.NotZero(t, person.Pets)
		assert.NotZero(t, person.FavoritePet)
	})
	t.Run("Panic Scenario", func(t *testing.T) {
		assert.Panics(t, func() {
			factory := fabricator.New[Person](Person{})
			_ = factory.Create()
		})
	})
}

func TestFactory_CreateBatch(t *testing.T) {
	t.Run("Success Scenario", func(t *testing.T) {
		var results []Person

		factory := fabricator.New[Person](Person{}, fabricator.Options[Person]{
			PersistenceHandler: func(instance Person) Person {
				results = append(results, instance)
				return instance
			},
		})

		people := factory.CreateBatch(5)
		assert.Len(t, people, 5)
		assert.Len(t, results, 5)
		assert.Equal(t, people, results)
	})
	t.Run("Panic Scenario", func(t *testing.T) {
		assert.Panics(t, func() {
			factory := fabricator.New[Person](Person{})
			_ = factory.CreateBatch(5)
		})
	})
}
