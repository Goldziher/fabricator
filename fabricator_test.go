package fabricator_test

import (
	"testing"
	"time"

	"github.com/Goldziher/fabricator"
	"github.com/stretchr/testify/assert"
)

type Pet struct {
	Name    string
	Species string
}

type Person struct {
	Id          int `faker:"oneof: 1, 2, 3, 4, 5, 6"`
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

type TestPersistenceHandler[T any] struct {
	ResultHandler func(...T)
}

func (handler TestPersistenceHandler[T]) Save(instance T) T {
	handler.ResultHandler(instance)
	return instance
}

func (handler TestPersistenceHandler[T]) SaveMany(instances []T) []T {
	handler.ResultHandler(instances...)
	return instances
}

func TestFactory_Create(t *testing.T) {
	t.Run("Success Scenario", func(t *testing.T) {
		var result Person
		handler := TestPersistenceHandler[Person]{ResultHandler: func(instances ...Person) {
			result = instances[0]
		}}
		factory := fabricator.New[Person](Person{}, fabricator.Options[Person]{
			PersistenceHandler: handler,
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
		handler := TestPersistenceHandler[Person]{ResultHandler: func(instances ...Person) {
			results = instances
		}}
		factory := fabricator.New[Person](Person{}, fabricator.Options[Person]{
			PersistenceHandler: handler,
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

func TestFactoryCounter(t *testing.T) {
	t.Run("Test Counter (regular)", func(t *testing.T) {
		factory := fabricator.New(Person{})
		for i := 0; i < 5; i++ {
			assert.Equal(t, i, factory.GetCounter())
			_ = factory.Build()
		}
	})
	t.Run("Test Counter (go routines)", func(t *testing.T) {
		factory := fabricator.New(Person{})
		for i := 0; i < 5; i++ {
			go func() { _ = factory.Build() }()
		}
		time.Sleep(time.Millisecond * 10)
		assert.Equal(t, 5, factory.GetCounter())
	})
	t.Run("Test Counter Reset", func(t *testing.T) {
		factory := fabricator.New(Person{})
		assert.Equal(t, 0, factory.GetCounter())
		_ = factory.Build()
		assert.Equal(t, 1, factory.GetCounter())
		factory.ResetCounter()
		assert.Equal(t, 0, factory.GetCounter())
	})
	t.Run("Test Set Counter", func(t *testing.T) {
		factory := fabricator.New(Person{})
		assert.Equal(t, 0, factory.GetCounter())
		factory.SetCounter(100)
		assert.Equal(t, 100, factory.GetCounter())
	})
}

func TestFactoryFunction(t *testing.T) {
	factory := fabricator.New(Person{}, fabricator.Options[Person]{
		Defaults: map[string]any{
			"Id": func(iteration int, fieldName string) interface{} {
				assert.Equal(t, "Id", fieldName)
				return iteration + 1
			},
		},
	})
	batch := factory.Batch(5)

	for i, person := range batch {
		assert.Equal(t, i+1, person.Id)
	}
}
