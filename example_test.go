package fabricator_test

import (
	"context"
	"fmt"

	"github.com/go-faker/faker/v4/pkg/options"

	"github.com/Goldziher/fabricator/v2"
)

func ExampleFactory_Build() {
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
	fmt.Println(person.ID, person.FirstName, person.LastName)
	// Output: 1 Moishe Zuchmir
}

func ExampleSubfactory() {
	type Pet struct {
		Name string
	}
	type Person struct {
		FavoritePet *Pet
		Pets        []Pet
	}

	petName := fabricator.FieldOf[Pet, string]("Name")
	petFactory := fabricator.New(Pet{}, fabricator.Value(petName, "Flippy"))
	personFactory := fabricator.New(
		Person{},
		fabricator.Field(fabricator.FieldOf[Person, *Pet]("FavoritePet"), fabricator.PtrSubfactory(petFactory)),
		fabricator.Field(fabricator.FieldOf[Person, []Pet]("Pets"), fabricator.SliceSubfactory(petFactory, 2)),
	)

	person := personFactory.Build()
	fmt.Println(person.FavoritePet.Name, len(person.Pets))
	// Output: Flippy 2
}

func ExampleWithFakerOptions() {
	type Person struct {
		Metadata any
	}

	factory := fabricator.New(
		Person{},
		fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
		fabricator.Value(fabricator.FieldOf[Person, any]("Metadata"), "configured"),
	)

	person := factory.Build()
	fmt.Println(person.Metadata)
	// Output: configured
}

func ExampleFactory_Create() {
	factory := fabricator.New(
		examplePerson{},
		fabricator.Value(fabricator.FieldOf[examplePerson, string]("Name"), "Moishe"),
		fabricator.WithPersistenceHandler[examplePerson](exampleHandler{}),
	)

	person := factory.Create(context.Background())
	fmt.Println(person.Name)
	// Output: Moishe
}

type exampleHandler struct{}

func (exampleHandler) Save(_ context.Context, person examplePerson) (examplePerson, error) {
	return person, nil
}

func (exampleHandler) SaveMany(_ context.Context, people []examplePerson) ([]examplePerson, error) {
	return people, nil
}

type examplePerson struct {
	Name string
}
