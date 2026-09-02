package fabricator_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Goldziher/fabricator/v2"
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
	Labels          map[string]string
}

type hiddenField struct {
	name string //nolint:unused
}

type EmbeddedOne struct {
	Code string
}

type EmbeddedTwo struct {
	Code string
}

type WithEmbedded struct {
	EmbeddedOne
	Name string
}

type WithAmbiguousEmbedded struct {
	EmbeddedOne
	EmbeddedTwo
}

func personFactory(opts ...fabricator.Option[Person]) *fabricator.Factory[Person] {
	base := []fabricator.Option[Person]{
		fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
	}

	return fabricator.New(Person{}, append(base, opts...)...)
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert.NotPanics(t, func() { _ = fabricator.New(Person{}) })
	})

	t.Run("rejects non-struct", func(t *testing.T) {
		assert.PanicsWithValue(t, "unsupported value: model must be a non-pointer struct", func() {
			_ = fabricator.New(100)
		})
	})

	t.Run("rejects pointer model", func(t *testing.T) {
		assert.PanicsWithValue(t, "unsupported value: model must be a non-pointer struct", func() {
			_ = fabricator.New(&Person{})
		})
	})

	t.Run("model value is type witness only", func(t *testing.T) {
		factory := fabricator.New(
			Person{FirstName: "template"},
			fabricator.WithFakerOptions[Person](options.WithIgnoreInterface(true)),
			fabricator.Value(fabricator.FieldOf[Person, int]("ID"), 1),
		)

		person := factory.Build()

		assert.NotEqual(t, "template", person.FirstName)
	})
}

func TestFieldOf(t *testing.T) {
	t.Run("validates field type", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = fabricator.FieldOf[Person, string]("FirstName")
		})
	})

	t.Run("rejects unknown field", func(t *testing.T) {
		assert.PanicsWithValue(t, `unknown field "Missing"`, func() {
			_ = fabricator.FieldOf[Person, string]("Missing")
		})
	})

	t.Run("rejects wrong field type", func(t *testing.T) {
		assert.PanicsWithValue(t, `field "ID" expects int, got string`, func() {
			_ = fabricator.FieldOf[Person, string]("ID")
		})
	})

	t.Run("rejects unexported field", func(t *testing.T) {
		assert.PanicsWithValue(t, `field "name" cannot be set`, func() {
			_ = fabricator.FieldOf[hiddenField, string]("name")
		})
	})

	t.Run("supports promoted embedded field", func(t *testing.T) {
		factory := fabricator.New(
			WithEmbedded{},
			fabricator.Value(fabricator.FieldOf[WithEmbedded, string]("Code"), "A1"),
		)

		value := factory.Build()

		assert.Equal(t, "A1", value.Code)
	})

	t.Run("rejects ambiguous embedded field", func(t *testing.T) {
		assert.PanicsWithValue(t, `unknown field "Code"`, func() {
			_ = fabricator.FieldOf[WithAmbiguousEmbedded, string]("Code")
		})
	})

	t.Run("rejects nested paths", func(t *testing.T) {
		assert.PanicsWithValue(t, `nested field paths are not supported: "EmbeddedOne.Code"`, func() {
			_ = fabricator.FieldOf[WithEmbedded, string]("EmbeddedOne.Code")
		})
	})
}

func TestFactoryBuild(t *testing.T) {
	t.Run("faker data", func(t *testing.T) {
		factory := personFactory()

		person := factory.Build()

		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.ID)
		assert.NotEmpty(t, person.FirstName)
		assert.NotEmpty(t, person.LastName)
		assert.NotZero(t, person.Pets)
		assert.NotZero(t, person.FavoritePet)
	})

	t.Run("typed defaults", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.FieldOf[Person, string]("FirstName"), "Moishe"),
			fabricator.Value(fabricator.FieldOf[Person, string]("LastName"), "Zuchmir"),
			fabricator.Value(fabricator.FieldOf[Person, []Pet]("Pets"), []Pet{{Name: "Flippy", Species: "Dolphin"}}),
		)

		person := factory.Build()

		assert.Equal(t, "Moishe", person.FirstName)
		assert.Equal(t, "Zuchmir", person.LastName)
		require.Len(t, person.Pets, 1)
		assert.Equal(t, "Flippy", person.Pets[0].Name)
		assert.Equal(t, "Dolphin", person.Pets[0].Species)
	})

	t.Run("typed build overrides", func(t *testing.T) {
		factory := personFactory()

		person := factory.Build(
			fabricator.Override(fabricator.FieldOf[Person, string]("FirstName"), "Moishe"),
			fabricator.Override(fabricator.FieldOf[Person, string]("LastName"), "Zuchmir"),
			fabricator.Override(fabricator.FieldOf[Person, []Pet]("Pets"), []Pet{{Name: "Flippy", Species: "Dolphin"}}),
		)

		assert.Equal(t, "Moishe", person.FirstName)
		assert.Equal(t, "Zuchmir", person.LastName)
		require.Len(t, person.Pets, 1)
		assert.Equal(t, "Flippy", person.Pets[0].Name)
		assert.Equal(t, "Dolphin", person.Pets[0].Species)
	})

	t.Run("unsafe field catches runtime type mismatch", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.UnsafeFieldOf[Person, string]("ID"), "not an int"),
		)

		_, err := factory.BuildE()

		require.EqualError(t, err, `field "ID" expects int, got string`)
	})
}

func TestNilAssignment(t *testing.T) {
	t.Run("supports nil interface", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.FieldOf[Person, any]("Metadata"), nil),
		)

		person := factory.Build()

		assert.Nil(t, person.Metadata)
	})

	t.Run("supports nil pointer", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.FieldOf[Person, *Profile]("FavoriteProfile"), nil),
		)

		person := factory.Build()

		assert.Nil(t, person.FavoriteProfile)
	})

	t.Run("supports nil slice", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.FieldOf[Person, []Pet]("Pets"), nil),
		)

		person := factory.Build()

		assert.Nil(t, person.Pets)
	})

	t.Run("supports nil map", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.FieldOf[Person, map[string]string]("Labels"), nil),
		)

		person := factory.Build()

		assert.Nil(t, person.Labels)
	})

	t.Run("rejects nil int", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.UnsafeFieldOf[Person, any]("ID"), nil),
		)

		_, err := factory.BuildE()

		require.EqualError(t, err, `field "ID" expects int, got nil`)
	})
}

func TestFactoryBatch(t *testing.T) {
	factory := personFactory(
		fabricator.Field(fabricator.FieldOf[Person, int]("ID"), func(ctx fabricator.BuildContext) int {
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

func TestFactoryBatchSize(t *testing.T) {
	factory := personFactory()

	empty := factory.Batch(0)
	negative, err := factory.BatchE(-1)

	assert.Empty(t, empty)
	assert.Nil(t, negative)
	require.EqualError(t, err, "batch size must be non-negative, got -1")
	assert.PanicsWithValue(t, "batch size must be non-negative, got -1", func() {
		_ = factory.Batch(-1)
	})
}

type TestPersistenceHandler[T any] struct {
	ResultHandler func(...T)
	SaveErr       error
	SaveManyErr   error
}

func (handler TestPersistenceHandler[T]) Save(_ context.Context, instance T) (T, error) {
	if handler.SaveErr != nil {
		var zero T
		return zero, handler.SaveErr
	}
	handler.ResultHandler(instance)
	return instance, nil
}

func (handler TestPersistenceHandler[T]) SaveMany(_ context.Context, instances []T) ([]T, error) {
	if handler.SaveManyErr != nil {
		return nil, handler.SaveManyErr
	}
	handler.ResultHandler(instances...)
	return instances, nil
}

func TestFactoryCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var result Person
		handler := TestPersistenceHandler[Person]{ResultHandler: func(instances ...Person) {
			result = instances[0]
		}}
		factory := personFactory(fabricator.WithPersistenceHandler[Person](handler))

		person := factory.Create(context.Background())

		assert.Equal(t, person, result)
		assert.IsType(t, Person{}, person)
		assert.NotZero(t, person.ID)
		assert.NotEmpty(t, person.FirstName)
	})

	t.Run("error without handler", func(t *testing.T) {
		factory := personFactory()

		_, err := factory.CreateE(context.Background())

		require.EqualError(t, err, "cannot call .Create on a factory without a persistence handler")
		assert.PanicsWithValue(t, "cannot call .Create on a factory without a persistence handler", func() {
			_ = factory.Create(context.Background())
		})
	})

	t.Run("propagates persistence error", func(t *testing.T) {
		factory := personFactory(fabricator.WithPersistenceHandler[Person](TestPersistenceHandler[Person]{
			ResultHandler: func(...Person) {},
			SaveErr:       errors.New("db down"),
		}))

		_, err := factory.CreateE(context.Background())

		require.EqualError(t, err, "persistence save failed: db down")
	})
}

func TestFactoryCreateBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var results []Person
		handler := TestPersistenceHandler[Person]{ResultHandler: func(instances ...Person) {
			results = instances
		}}
		factory := personFactory(fabricator.WithPersistenceHandler[Person](handler))

		people := factory.CreateBatch(context.Background(), 5)

		assert.Len(t, people, 5)
		assert.Len(t, results, 5)
		assert.Equal(t, people, results)
	})

	t.Run("propagates persistence error", func(t *testing.T) {
		factory := personFactory(fabricator.WithPersistenceHandler[Person](TestPersistenceHandler[Person]{
			ResultHandler: func(...Person) {},
			SaveManyErr:   errors.New("db down"),
		}))

		_, err := factory.CreateBatchE(context.Background(), 2)

		require.EqualError(t, err, "persistence save many failed: db down")
	})
}

func TestFactoryCounter(t *testing.T) {
	t.Run("regular", func(t *testing.T) {
		factory := personFactory()
		for i := range 5 {
			assert.Equal(t, i, factory.GetCounter())
			_ = factory.Build()
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		factory := personFactory()
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
		factory := personFactory()
		assert.Equal(t, 0, factory.GetCounter())
		_ = factory.Build()
		assert.Equal(t, 1, factory.GetCounter())
		factory.ResetCounter()
		assert.Equal(t, 0, factory.GetCounter())
	})

	t.Run("set", func(t *testing.T) {
		factory := personFactory()
		assert.Equal(t, 0, factory.GetCounter())
		factory.SetCounter(100)
		assert.Equal(t, 100, factory.GetCounter())
	})

	t.Run("setter error consumes iteration", func(t *testing.T) {
		factory := personFactory(
			fabricator.Value(fabricator.UnsafeFieldOf[Person, string]("ID"), "bad"),
		)

		_, err := factory.BuildE()

		require.Error(t, err)
		assert.Equal(t, 1, factory.GetCounter())
	})
}

func TestFieldProviderContext(t *testing.T) {
	factory := personFactory(
		fabricator.Field(fabricator.FieldOf[Person, int]("ID"), func(ctx fabricator.BuildContext) int {
			assert.Equal(t, "ID", ctx.FieldName)
			return ctx.Iteration + 1
		}),
	)

	batch := factory.Batch(5)

	for i, person := range batch {
		assert.Equal(t, i+1, person.ID)
	}
}

func TestLifecycleHooks(t *testing.T) {
	factory := personFactory(
		fabricator.AfterFaker[Person](func(person *Person, ctx fabricator.BuildContext) error {
			person.FirstName = fmt.Sprintf("first-%d", ctx.Iteration)
			return nil
		}),
		fabricator.AfterBuild[Person](func(person *Person, _ fabricator.BuildContext) error {
			person.LastName = person.FirstName + "-last"
			return nil
		}),
		fabricator.AfterCreate[Person](func(person *Person, _ fabricator.BuildContext) error {
			person.Metadata = "created"
			return nil
		}),
		fabricator.WithPersistenceHandler[Person](TestPersistenceHandler[Person]{
			ResultHandler: func(...Person) {},
		}),
	)

	built := factory.Build()
	created := factory.Create(context.Background())

	assert.Equal(t, "first-0", built.FirstName)
	assert.Equal(t, "first-0-last", built.LastName)
	assert.Equal(t, "created", created.Metadata)
}

func TestAfterCreateReceivesBuildIteration(t *testing.T) {
	iterations := []int{}
	factory := personFactory(
		fabricator.WithPersistenceHandler[Person](TestPersistenceHandler[Person]{
			ResultHandler: func(...Person) {},
		}),
		fabricator.AfterCreate[Person](func(_ *Person, ctx fabricator.BuildContext) error {
			iterations = append(iterations, ctx.Iteration)
			return nil
		}),
	)

	_ = factory.Create(context.Background())
	_ = factory.CreateBatch(context.Background(), 2)

	assert.Equal(t, []int{0, 1, 2}, iterations)
}

func TestLifecycleHookErrors(t *testing.T) {
	t.Run("after faker", func(t *testing.T) {
		factory := personFactory(fabricator.AfterFaker[Person](func(*Person, fabricator.BuildContext) error {
			return errors.New("bad faker")
		}))

		_, err := factory.BuildE()

		require.EqualError(t, err, "after faker hook failed: bad faker")
	})

	t.Run("after build", func(t *testing.T) {
		factory := personFactory(fabricator.AfterBuild[Person](func(*Person, fabricator.BuildContext) error {
			return errors.New("bad build")
		}))

		_, err := factory.BuildE()

		require.EqualError(t, err, "after build hook failed: bad build")
	})

	t.Run("after create", func(t *testing.T) {
		factory := personFactory(
			fabricator.WithPersistenceHandler[Person](TestPersistenceHandler[Person]{ResultHandler: func(...Person) {}}),
			fabricator.AfterCreate[Person](func(*Person, fabricator.BuildContext) error {
				return errors.New("bad create")
			}),
		)

		_, err := factory.CreateE(context.Background())

		require.EqualError(t, err, "after create hook failed: bad create")
	})
}

func TestSubfactories(t *testing.T) {
	petFactory := fabricator.New(
		Pet{},
		fabricator.Value(fabricator.FieldOf[Pet, string]("Name"), "Flippy"),
		fabricator.Value(fabricator.FieldOf[Pet, string]("Species"), "Dolphin"),
	)
	profileFactory := fabricator.New(Profile{}, fabricator.Value(fabricator.FieldOf[Profile, string]("Reason"), "friendly"))
	personFactory := personFactory(
		fabricator.Field(fabricator.FieldOf[Person, Pet]("FavoritePet"), fabricator.Subfactory(petFactory)),
		fabricator.Field(fabricator.FieldOf[Person, *Profile]("FavoriteProfile"), fabricator.PtrSubfactory(profileFactory)),
		fabricator.Field(fabricator.FieldOf[Person, []Pet]("Pets"), fabricator.SliceSubfactory(petFactory, 2)),
	)

	person := personFactory.Build()

	assert.Equal(t, "Flippy", person.FavoritePet.Name)
	assert.Equal(t, "Dolphin", person.FavoritePet.Species)
	require.NotNil(t, person.FavoriteProfile)
	assert.Equal(t, "friendly", person.FavoriteProfile.Reason)
	require.Len(t, person.Pets, 2)
	assert.Equal(t, "Flippy", person.Pets[0].Name)
}

func TestContextAwareSubfactories(t *testing.T) {
	petName := fabricator.FieldOf[Pet, string]("Name")
	petFactory := fabricator.New(Pet{})
	personFactory := personFactory(
		fabricator.Field(fabricator.FieldOf[Person, Pet]("FavoritePet"), fabricator.SubfactoryWith(
			petFactory,
			func(ctx fabricator.BuildContext) []fabricator.BuildOption[Pet] {
				return []fabricator.BuildOption[Pet]{
					fabricator.Override(petName, fmt.Sprintf("pet-%d", ctx.Iteration)),
				}
			},
		)),
		fabricator.Field(fabricator.FieldOf[Person, []Pet]("Pets"), fabricator.SliceSubfactoryWith(
			petFactory,
			func(ctx fabricator.BuildContext) int { return ctx.Iteration + 1 },
			func(ctx fabricator.BuildContext) []fabricator.BuildOption[Pet] {
				return []fabricator.BuildOption[Pet]{
					fabricator.Override(petName, fmt.Sprintf("batch-%d", ctx.Iteration)),
				}
			},
		)),
	)

	first := personFactory.Build()
	second := personFactory.Build()

	assert.Equal(t, "pet-0", first.FavoritePet.Name)
	require.Len(t, first.Pets, 1)
	assert.Equal(t, "batch-0", first.Pets[0].Name)
	assert.Equal(t, "pet-1", second.FavoritePet.Name)
	require.Len(t, second.Pets, 2)
	assert.Equal(t, "batch-1", second.Pets[0].Name)
}

func TestSubfactoryPanicsOnNilFactory(t *testing.T) {
	assert.PanicsWithValue(t, "subfactory cannot use a nil factory", func() {
		_ = fabricator.Subfactory[Pet](nil)
	})
	assert.PanicsWithValue(t, "subfactory cannot use a nil factory", func() {
		_ = fabricator.PtrSubfactory[Pet](nil)
	})
	assert.PanicsWithValue(t, "subfactory cannot use a nil factory", func() {
		_ = fabricator.SliceSubfactory[Pet](nil, 1)
	})
	assert.PanicsWithValue(t, "subfactory size function cannot be nil", func() {
		_ = fabricator.SliceSubfactoryWith(fabricator.New(Pet{}), nil, nil)
	})
}

func TestFakerOptions(t *testing.T) {
	factory := personFactory(
		fabricator.Value(fabricator.FieldOf[Person, any]("Metadata"), "configured"),
	)

	person := factory.Build()

	assert.Equal(t, "configured", person.Metadata)
}

func TestWithoutFaker(t *testing.T) {
	t.Run("leaves unconfigured fields at their zero value", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Value(fabricator.FieldOf[Person, string]("FirstName"), "Moishe"),
		)

		person := factory.Build()

		assert.Equal(t, "Moishe", person.FirstName)
		assert.Empty(t, person.LastName)
		assert.Zero(t, person.ID)
		assert.Nil(t, person.Pets)
		assert.Nil(t, person.Labels)
		assert.Nil(t, person.FavoriteProfile)
		assert.Equal(t, Pet{}, person.FavoritePet)
	})

	t.Run("still runs after faker hooks against the zero value", func(t *testing.T) {
		var seen Person
		factory := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.AfterFaker(func(person *Person, _ fabricator.BuildContext) error {
				seen = *person
				person.LastName = "from hook"

				return nil
			}),
		)

		person := factory.Build()

		assert.Equal(t, Person{}, seen)
		assert.Equal(t, "from hook", person.LastName)
	})

	t.Run("still advances the counter", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithoutFaker[Person]())

		factory.Batch(3)

		assert.Equal(t, 3, factory.GetCounter())
	})

	t.Run("does not generate an interface field faker would reject", func(t *testing.T) {
		// Person.Metadata is why personFactory needs WithIgnoreInterface; skipping
		// faker has to sidestep that failure rather than trip over it.
		factory := fabricator.New(Person{}, fabricator.WithoutFaker[Person]())

		person, err := factory.BuildE()

		require.NoError(t, err)
		assert.Nil(t, person.Metadata)
	})
}

func TestSequence(t *testing.T) {
	t.Run("cycles through values by iteration", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Field(
				fabricator.FieldOf[Person, string]("FirstName"),
				fabricator.Sequence("a", "b", "c"),
			),
		)

		people := factory.Batch(5)

		names := make([]string, 0, len(people))
		for _, person := range people {
			names = append(names, person.FirstName)
		}
		assert.Equal(t, []string{"a", "b", "c", "a", "b"}, names)
	})

	t.Run("handles a single value", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Field(fabricator.FieldOf[Person, string]("FirstName"), fabricator.Sequence("only")),
		)

		for _, person := range factory.Batch(3) {
			assert.Equal(t, "only", person.FirstName)
		}
	})

	t.Run("stays in range for a negative counter", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Field(
				fabricator.FieldOf[Person, string]("FirstName"),
				fabricator.Sequence("a", "b", "c"),
			),
		)
		factory.SetCounter(-4)

		people := factory.Batch(3)

		// Iterations -4, -3, -2 must wrap forwards, not index out of range.
		names := make([]string, 0, len(people))
		for _, person := range people {
			names = append(names, person.FirstName)
		}
		assert.Equal(t, []string{"c", "a", "b"}, names)
	})

	t.Run("advances in lockstep across fields", func(t *testing.T) {
		factory := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Field(fabricator.FieldOf[Person, string]("FirstName"), fabricator.Sequence("a", "b")),
			fabricator.Field(fabricator.FieldOf[Person, string]("LastName"), fabricator.Sequence("x", "y")),
		)

		people := factory.Batch(2)

		assert.Equal(t, "a", people[0].FirstName)
		assert.Equal(t, "x", people[0].LastName)
		assert.Equal(t, "b", people[1].FirstName)
		assert.Equal(t, "y", people[1].LastName)
	})

	t.Run("works as a build override", func(t *testing.T) {
		factory := fabricator.New(Person{}, fabricator.WithoutFaker[Person]())

		people := factory.Batch(
			3,
			fabricator.OverrideField(
				fabricator.FieldOf[Person, string]("FirstName"),
				fabricator.Sequence("a", "b"),
			),
		)

		assert.Equal(t, "a", people[0].FirstName)
		assert.Equal(t, "b", people[1].FirstName)
		assert.Equal(t, "a", people[2].FirstName)
	})

	t.Run("panics without values", func(t *testing.T) {
		assert.PanicsWithValue(t, "sequence requires at least one value", func() {
			fabricator.Sequence[string]()
		})
	})
}

func TestExtend(t *testing.T) {
	firstName := fabricator.FieldOf[Person, string]("FirstName")
	lastName := fabricator.FieldOf[Person, string]("LastName")

	t.Run("inherits the base configuration", func(t *testing.T) {
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Value(firstName, "Moishe"),
			fabricator.Value(lastName, "Zuchmir"),
		)

		derived := fabricator.Extend(base)

		person := derived.Build()
		assert.Equal(t, "Moishe", person.FirstName)
		assert.Equal(t, "Zuchmir", person.LastName)
	})

	t.Run("later options win over inherited ones", func(t *testing.T) {
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Value(firstName, "Moishe"),
			fabricator.Value(lastName, "Zuchmir"),
		)

		derived := fabricator.Extend(base, fabricator.Value(lastName, "Ben Gurion"))

		person := derived.Build()
		assert.Equal(t, "Moishe", person.FirstName)
		assert.Equal(t, "Ben Gurion", person.LastName)
	})

	t.Run("does not mutate the base factory", func(t *testing.T) {
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Value(firstName, "Moishe"),
		)

		fabricator.Extend(base, fabricator.Value(lastName, "Derived"))

		person := base.Build()
		assert.Equal(t, "Moishe", person.FirstName)
		assert.Empty(t, person.LastName, "extending must not add fields to the base factory")
	})

	t.Run("sibling factories do not share a backing array", func(t *testing.T) {
		// Three defaults leave the base slice with spare capacity, which is the
		// only condition under which the bug this guards against can appear: if
		// Extend handed the derived factory base's slice as-is, both siblings
		// would append into the same spare slot and the second would overwrite
		// the first. Two defaults would grow exactly to cap and reallocate on
		// append, hiding the aliasing.
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.Value(firstName, "Moishe"),
			fabricator.Value(fabricator.FieldOf[Person, int]("ID"), 7),
			fabricator.Value(fabricator.FieldOf[Person, any]("Metadata"), "shared"),
		)

		first := fabricator.Extend(base, fabricator.Value(lastName, "First"))
		second := fabricator.Extend(base, fabricator.Value(lastName, "Second"))

		assert.Equal(t, "First", first.Build().LastName)
		assert.Equal(t, "Second", second.Build().LastName)
		assert.Equal(t, "Moishe", first.Build().FirstName, "inherited defaults must survive")
	})

	t.Run("sibling factories do not share a hook backing array", func(t *testing.T) {
		// Same spare-capacity condition as above, for the hook slices.
		var order []string
		record := func(label string) fabricator.Hook[Person] {
			return func(*Person, fabricator.BuildContext) error {
				order = append(order, label)

				return nil
			}
		}
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.AfterBuild(record("base1")),
			fabricator.AfterBuild(record("base2")),
			fabricator.AfterBuild(record("base3")),
		)

		first := fabricator.Extend(base, fabricator.AfterBuild(record("first")))
		second := fabricator.Extend(base, fabricator.AfterBuild(record("second")))

		order = nil
		first.Build()
		assert.Equal(t, []string{"base1", "base2", "base3", "first"}, order)

		order = nil
		second.Build()
		assert.Equal(t, []string{"base1", "base2", "base3", "second"}, order)
	})

	t.Run("hooks are additive and ordered base first", func(t *testing.T) {
		var order []string
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.AfterBuild(func(*Person, fabricator.BuildContext) error {
				order = append(order, "base")

				return nil
			}),
		)

		derived := fabricator.Extend(base, fabricator.AfterBuild(func(*Person, fabricator.BuildContext) error {
			order = append(order, "derived")

			return nil
		}))

		derived.Build()
		assert.Equal(t, []string{"base", "derived"}, order)

		order = nil
		base.Build()
		assert.Equal(t, []string{"base"}, order, "the derived hook must not run on the base factory")
	})

	t.Run("counters are independent and start at zero", func(t *testing.T) {
		base := fabricator.New(Person{}, fabricator.WithoutFaker[Person]())
		base.Batch(3)

		derived := fabricator.Extend(base)

		assert.Equal(t, 0, derived.GetCounter())
		derived.Build()
		assert.Equal(t, 1, derived.GetCounter())
		assert.Equal(t, 3, base.GetCounter(), "building the derived factory must not advance the base counter")
	})

	t.Run("inherits the persistence handler", func(t *testing.T) {
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.WithPersistenceHandler[Person](&recordingHandler{}),
		)

		derived := fabricator.Extend(base, fabricator.Value(firstName, "Moishe"))

		person, err := derived.CreateE(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "Moishe", person.FirstName)
	})

	t.Run("a handler in opts replaces the inherited one", func(t *testing.T) {
		baseHandler := &recordingHandler{}
		derivedHandler := &recordingHandler{}
		base := fabricator.New(
			Person{},
			fabricator.WithoutFaker[Person](),
			fabricator.WithPersistenceHandler[Person](baseHandler),
		)

		derived := fabricator.Extend(base, fabricator.WithPersistenceHandler[Person](derivedHandler))

		_, err := derived.CreateE(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1, derivedHandler.saved)
		assert.Equal(t, 0, baseHandler.saved)
	})

	t.Run("inherits faker configuration", func(t *testing.T) {
		base := personFactory()

		derived := fabricator.Extend(base, fabricator.Value(firstName, "Moishe"))

		// Without the inherited WithIgnoreInterface, faker fails on Person.Metadata.
		person, err := derived.BuildE()
		require.NoError(t, err)
		assert.Equal(t, "Moishe", person.FirstName)
	})

	t.Run("inherits a skipped faker and can be extended further", func(t *testing.T) {
		base := fabricator.New(Person{}, fabricator.WithoutFaker[Person]())

		derived := fabricator.Extend(fabricator.Extend(base), fabricator.Value(firstName, "Moishe"))

		person := derived.Build()
		assert.Equal(t, "Moishe", person.FirstName)
		assert.Empty(t, person.LastName)
	})

	t.Run("panics on a nil factory", func(t *testing.T) {
		assert.PanicsWithValue(t, "cannot extend a nil factory", func() {
			fabricator.Extend[Person](nil)
		})
	})
}

type recordingHandler struct {
	saved int
}

func (handler *recordingHandler) Save(_ context.Context, person Person) (Person, error) {
	handler.saved++

	return person, nil
}

func (handler *recordingHandler) SaveMany(_ context.Context, people []Person) ([]Person, error) {
	handler.saved += len(people)

	return people, nil
}

func TestSeed(t *testing.T) {
	// Seed sets faker's process-wide source, so these subtests must not run in
	// parallel with anything that builds from faker.
	build := func() Person {
		return personFactory().Build()
	}

	t.Run("the same seed reproduces the same data", func(t *testing.T) {
		fabricator.Seed(42)
		first := build()

		fabricator.Seed(42)
		second := build()

		assert.Equal(t, first, second)
	})

	t.Run("a different seed produces different data", func(t *testing.T) {
		fabricator.Seed(42)
		first := build()

		fabricator.Seed(43)
		second := build()

		assert.NotEqual(t, first, second)
	})

	t.Run("reproduces a whole batch, not just the first build", func(t *testing.T) {
		fabricator.Seed(7)
		first := personFactory().Batch(5)

		fabricator.Seed(7)
		second := personFactory().Batch(5)

		assert.Equal(t, first, second)
	})

	t.Run("generated data is still populated", func(t *testing.T) {
		fabricator.Seed(1)

		person := build()

		assert.NotEmpty(t, person.FirstName, "seeding must not stop faker from generating")
	})
}

func BenchmarkBuild(b *testing.B) {
	factory := personFactory()

	b.ReportAllocs()

	for b.Loop() {
		_ = factory.Build()
	}
}

func BenchmarkBuildWithoutFaker(b *testing.B) {
	factory := fabricator.New(
		Person{},
		fabricator.WithoutFaker[Person](),
		fabricator.Value(fabricator.FieldOf[Person, string]("FirstName"), "Moishe"),
		fabricator.Value(fabricator.FieldOf[Person, string]("LastName"), "Zuchmir"),
	)

	b.ReportAllocs()

	for b.Loop() {
		_ = factory.Build()
	}
}

func BenchmarkBatch(b *testing.B) {
	factory := personFactory()

	b.ReportAllocs()

	for b.Loop() {
		_ = factory.Batch(10)
	}
}

func BenchmarkExtend(b *testing.B) {
	base := personFactory(
		fabricator.Value(fabricator.FieldOf[Person, string]("FirstName"), "Moishe"),
	)
	lastName := fabricator.FieldOf[Person, string]("LastName")

	b.ReportAllocs()

	for b.Loop() {
		_ = fabricator.Extend(base, fabricator.Value(lastName, "Zuchmir"))
	}
}
