package fabricator_test

import (
	"github.com/Goldziher/fabricator"
	"github.com/stretchr/testify/assert"
	"testing"
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
		assert.NotPanics(t, func() { _ = fabricator.New(Person{}, nil) })
	})
	t.Run("Failure Scenario", func(t *testing.T) {
		assert.Panics(t, func() { _ = fabricator.New(100, nil) })
	})
}

func TestFactoryBuild(t *testing.T) {
	factory := fabricator.New(Person{}, nil)
	person := factory.Build()
	assert.IsType(t, Person{}, person)
}
