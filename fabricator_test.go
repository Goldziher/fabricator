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

func TestFactoryBuild(t *testing.T) {
	factory := fabricator.New(Person{})
	person := factory.Build()
	assert.IsType(t, Person{}, person)
	assert.NotZero(t, person.Id)
	assert.NotZero(t, person.FirstName)
	assert.NotZero(t, person.LastName)
	assert.NotZero(t, person.Pets)
	assert.NotZero(t, person.FavoritePet)
}

func TestFactoryBuildWithOverrides(t *testing.T) {
	factory := fabricator.New(Person{})
	person := factory.BuildWithOverrides(map[string]interface{}{
		"FirstName": "Moishe",
		"LastName":  "Zuchmir",
		"Pets": []Pet{
			Pet{
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
}
