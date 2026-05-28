package fabricator

import (
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
)

// BuildContext contains metadata for the value currently being built.
type BuildContext struct {
	Iteration int
	FieldName string
}

// FieldProvider returns a typed field value for a generated instance.
type FieldProvider[V any] func(BuildContext) V

// Option configures a Factory.
type Option[T any] func(*Factory[T])

// BuildOption configures a single Build, Create, Batch, or CreateBatch call.
type BuildOption[T any] func(*buildConfig[T])

// PersistenceHandler persists generated instances.
type PersistenceHandler[T any] interface {
	Save(instance T) T
	SaveMany(instance []T) []T
}

type fieldSetter[T any] struct {
	name    string
	provide func(BuildContext) any
}

type buildConfig[T any] struct {
	fields []fieldSetter[T]
}

// Factory builds typed test data instances.
type Factory[T any] struct {
	model              T
	persistenceHandler PersistenceHandler[T]
	defaults           []fieldSetter[T]
	fakerOptions       []options.OptionFunc
	counter            atomic.Int64
}

// New creates a factory for a struct of type T.
func New[T any](model T, opts ...Option[T]) *Factory[T] {
	modelType := reflect.TypeOf(model)
	if modelType == nil || modelType.Kind() != reflect.Struct {
		panic("unsupported value: model must be a struct")
	}

	factory := &Factory[T]{
		model: model,
	}
	for _, opt := range opts {
		opt(factory)
	}

	return factory
}

// WithPersistenceHandler configures persistence for Create and CreateBatch.
func WithPersistenceHandler[T any](handler PersistenceHandler[T]) Option[T] {
	return func(factory *Factory[T]) {
		factory.persistenceHandler = handler
	}
}

// WithFakerOptions configures github.com/go-faker/faker/v4 generation.
func WithFakerOptions[T any](opts ...options.OptionFunc) Option[T] {
	return func(factory *Factory[T]) {
		factory.fakerOptions = append(factory.fakerOptions, opts...)
	}
}

// Field configures a typed provider for a struct field.
func Field[T any, V any](name string, provider FieldProvider[V]) Option[T] {
	setter := newFieldSetter[T](name, provider)

	return func(factory *Factory[T]) {
		factory.defaults = append(factory.defaults, setter)
	}
}

// Value configures a static typed value for a struct field.
func Value[T any, V any](name string, value V) Option[T] {
	return Field[T](name, func(BuildContext) V {
		return value
	})
}

// Override configures a typed field value for a single build call.
func Override[T any, V any](name string, value V) BuildOption[T] {
	return OverrideField[T](name, func(BuildContext) V {
		return value
	})
}

// OverrideField configures a typed field provider for a single build call.
func OverrideField[T any, V any](name string, provider FieldProvider[V]) BuildOption[T] {
	setter := newFieldSetter[T](name, provider)

	return func(config *buildConfig[T]) {
		config.fields = append(config.fields, setter)
	}
}

// Subfactory builds a nested struct value from another factory.
func Subfactory[T any](factory *Factory[T]) FieldProvider[T] {
	return func(BuildContext) T {
		return factory.Build()
	}
}

// PtrSubfactory builds a nested struct pointer from another factory.
func PtrSubfactory[T any](factory *Factory[T]) FieldProvider[*T] {
	return func(BuildContext) *T {
		value := factory.Build()
		return &value
	}
}

// SliceSubfactory builds a slice of nested structs from another factory.
func SliceSubfactory[T any](factory *Factory[T], size int) FieldProvider[[]T] {
	return func(BuildContext) []T {
		return factory.Batch(size)
	}
}

func newFieldSetter[T any, V any](name string, provider FieldProvider[V]) fieldSetter[T] {
	return fieldSetter[T]{
		name: name,
		provide: func(ctx BuildContext) any {
			return provider(ctx)
		},
	}
}

func applyFieldSetter[T any](model *T, iteration int, setter fieldSetter[T]) {
	field := reflect.ValueOf(model).Elem().FieldByName(setter.name)
	if !field.IsValid() {
		panic(fmt.Sprintf("unknown field %q", setter.name))
	}
	if !field.CanSet() {
		panic(fmt.Sprintf("field %q cannot be set", setter.name))
	}

	value := reflect.ValueOf(setter.provide(BuildContext{
		Iteration: iteration,
		FieldName: setter.name,
	}))
	if !value.IsValid() {
		panic(fmt.Sprintf("field %q received invalid nil value", setter.name))
	}

	if value.Type().AssignableTo(field.Type()) {
		field.Set(value)
		return
	}
	if value.Type().ConvertibleTo(field.Type()) {
		field.Set(value.Convert(field.Type()))
		return
	}

	panic(fmt.Sprintf(
		"field %q expects %s, got %s",
		setter.name,
		field.Type(),
		value.Type(),
	))
}

// ResetCounter resets the factory internal counter to 0.
func (factory *Factory[T]) ResetCounter() {
	factory.counter.Store(0)
}

// GetCounter returns the current value of the factory's internal counter.
func (factory *Factory[T]) GetCounter() int {
	return int(factory.counter.Load())
}

// SetCounter sets the factory's counter to the specified value.
func (factory *Factory[T]) SetCounter(value int) {
	factory.counter.Store(int64(value))
}

// Build creates an instance of the factory's model struct.
func (factory *Factory[T]) Build(overrides ...BuildOption[T]) T {
	var model T
	if fakerErr := faker.FakeData(&model, factory.fakerOptions...); fakerErr != nil {
		panic(fmt.Errorf("error generating fake data: %w", fakerErr).Error())
	}

	iteration := int(factory.counter.Add(1) - 1)
	for _, setter := range factory.defaults {
		applyFieldSetter(&model, iteration, setter)
	}

	config := buildConfig[T]{}
	for _, override := range overrides {
		override(&config)
	}
	for _, setter := range config.fields {
		applyFieldSetter(&model, iteration, setter)
	}

	return model
}

// Batch builds a slice of the factory's model of a given size.
func (factory *Factory[T]) Batch(size int, overrides ...BuildOption[T]) []T {
	batch := make([]T, 0, size)
	for range size {
		batch = append(batch, factory.Build(overrides...))
	}
	return batch
}

// Create builds an instance and persists it using the factory's persistence handler.
func (factory *Factory[T]) Create(overrides ...BuildOption[T]) T {
	if factory.persistenceHandler == nil {
		panic("cannot call .Create on a factory without a persistence handler")
	}
	instance := factory.Build(overrides...)
	return factory.persistenceHandler.Save(instance)
}

// CreateBatch builds a batch and persists it using the factory's persistence handler.
func (factory *Factory[T]) CreateBatch(size int, overrides ...BuildOption[T]) []T {
	if factory.persistenceHandler == nil {
		panic("cannot call .CreateBatch on a factory without a persistence handler")
	}
	batch := factory.Batch(size, overrides...)
	return factory.persistenceHandler.SaveMany(batch)
}
