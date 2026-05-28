package fabricator_test

import (
	"sync"
	"testing"

	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Goldziher/fabricator"
)

type Pet struct {
	Name    string
	Species string
}

type Profile struct {
	Reason string
}

type Person struct {
	ID              int `faker:"oneof: 1, 2, 3, 4, 5, 6"`
	FirstName       string
	LastName        string
	Pets            []Pet
	FavoritePet     Pet
	FavoriteProfile *Profile
	Metadata        any
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert.NotPanics(t, func() { _ = fabricator.New(Person{}) })
	})

	t.Run("failure", func(t *testing.T) {
		assert.Panics(t, func() { _ = fabricator.New(100) })
	})
}

func TestFactoryBuild(t *testing.T) {
	t.Run("faker data", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))

		person := factory.Build()

		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.ID)
		assert.NotEmpty(t, person.FirstName)
		assert.NotEmpty(t, person.LastName)
		assert.NotZero(t, person.Pets)
		assert.NotZero(t, person.FavoritePet)
	})

	t.Run("typed defaults", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
			fabricator.Value[Person]("FirstName", "Moishe"),
			fabricator.Value[Person]("LastName", "Zuchmir"),
			fabricator.Value[Person]("Pets", []Pet{{Name: "Flippy", Species: "Dolphin"}}),
		)

		person := factory.Build()

		assert.Equal(t, "Moishe", person.FirstName)
		assert.Equal(t, "Zuchmir", person.LastName)
		require.Len(t, person.Pets, 1)
		assert.Equal(t, "Flippy", person.Pets[0].Name)
		assert.Equal(t, "Dolphin", person.Pets[0].Species)
	})

	t.Run("typed build overrides", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))

		person := factory.Build(
			fabricator.Override[Person]("FirstName", "Moishe"),
			fabricator.Override[Person]("LastName", "Zuchmir"),
			fabricator.Override[Person]("Pets", []Pet{{Name: "Flippy", Species: "Dolphin"}}),
		)

		assert.Equal(t, "Moishe", person.FirstName)
		assert.Equal(t, "Zuchmir", person.LastName)
		require.Len(t, person.Pets, 1)
		assert.Equal(t, "Flippy", person.Pets[0].Name)
		assert.Equal(t, "Dolphin", person.Pets[0].Species)
	})
}

func TestFactoryBatch(t *testing.T) {
	factory := fabricator.New(
		Person{},
		fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
		fabricator.Field[Person]("ID", func(ctx fabricator.BuildContext) int {
			return ctx.Iteration + 1
		}),
	)

	people := factory.Batch(5)

	require.Len(t, people, 5)
	for i, person := range people {
		assert.Equal(t, i+1, person.ID)
		assert.NotEmpty(t, person.FirstName)
	}
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

func TestFactoryCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var result Person
		handler := TestPersistenceHandler[Person]{ResultHandler: func(instances ...Person) {
			result = instances[0]
		}}
		factory := fabricator.New(
			Person{},
			fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
			fabricator.WithPersistenceHandler[Person](handler),
		)

		person := factory.Create()

		assert.Equal(t, person, result)
		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.ID)
		assert.NotEmpty(t, person.FirstName)
	})

	t.Run("panic without handler", func(t *testing.T) {
		assert.Panics(t, func() {
			factory := fabricator.New[Person](Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))
			_ = factory.Create()
		})
	})
}

func TestFactoryCreateBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var results []Person
		handler := TestPersistenceHandler[Person]{ResultHandler: func(instances ...Person) {
			results = instances
		}}
		factory := fabricator.New(
			Person{},
			fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
			fabricator.WithPersistenceHandler[Person](handler),
		)

		people := factory.CreateBatch(5)

		assert.Len(t, people, 5)
		assert.Len(t, results, 5)
		assert.Equal(t, people, results)
	})

	t.Run("panic without handler", func(t *testing.T) {
		assert.Panics(t, func() {
			factory := fabricator.New[Person](Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))
			_ = factory.CreateBatch(5)
		})
	})
}

func TestFactoryCounter(t *testing.T) {
	t.Run("regular", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))
		for i := range 5 {
			assert.Equal(t, i, factory.GetCounter())
			_ = factory.Build()
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))
		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				_ = factory.Build()
			})
		}
		wg.Wait()

		assert.Equal(t, 5, factory.GetCounter())
	})

	t.Run("reset", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))
		assert.Equal(t, 0, factory.GetCounter())
		_ = factory.Build()
		assert.Equal(t, 1, factory.GetCounter())
		factory.ResetCounter()
		assert.Equal(t, 0, factory.GetCounter())
	})

	t.Run("set", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)))
		assert.Equal(t, 0, factory.GetCounter())
		factory.SetCounter(100)
		assert.Equal(t, 100, factory.GetCounter())
	})
}

func TestFieldProviderContext(t *testing.T) {
	factory := fabricator.New(
		Person{},
		fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
		fabricator.Field[Person]("ID", func(ctx fabricator.BuildContext) int {
			assert.Equal(t, "ID", ctx.FieldName)
			return ctx.Iteration + 1
		}),
	)

	batch := factory.Batch(5)

	for i, person := range batch {
		assert.Equal(t, i+1, person.ID)
	}
}

func TestSubfactories(t *testing.T) {
	petFactory := fabricator.New(
		Pet{},
		fabricator.Value[Pet]("Name", "Flippy"),
		fabricator.Value[Pet]("Species", "Dolphin"),
	)
	profileFactory := fabricator.New(Profile{}, fabricator.Value[Profile]("Reason", "friendly"))
	personFactory := fabricator.New(
		Person{},
		fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
		fabricator.Field[Person]("FavoritePet", fabricator.Subfactory(petFactory)),
		fabricator.Field[Person]("FavoriteProfile", fabricator.PtrSubfactory(profileFactory)),
		fabricator.Field[Person]("Pets", fabricator.SliceSubfactory(petFactory, 2)),
	)

	person := personFactory.Build()

	assert.Equal(t, "Flippy", person.FavoritePet.Name)
	assert.Equal(t, "Dolphin", person.FavoritePet.Species)
	require.NotNil(t, person.FavoriteProfile)
	assert.Equal(t, "friendly", person.FavoriteProfile.Reason)
	require.Len(t, person.Pets, 2)
	assert.Equal(t, "Flippy", person.Pets[0].Name)
}

func TestFakerOptions(t *testing.T) {
	factory := fabricator.New(
		Person{},
		fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
		fabricator.Value[Person]("Metadata", "configured"),
	)

	person := factory.Build()

	assert.Equal(t, "configured", person.Metadata)
}

func TestInvalidFieldPanics(t *testing.T) {
	t.Run("unknown field", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
			fabricator.Value[Person]("Missing", "value"),
		)

		assert.PanicsWithValue(t, `unknown field "Missing"`, func() {
			_ = factory.Build()
		})
	})

	t.Run("wrong type", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
			fabricator.Value[Person]("ID", "not an int"),
		)

		assert.PanicsWithValue(t, `field "ID" expects int, got string`, func() {
			_ = factory.Build()
		})
	})
}
