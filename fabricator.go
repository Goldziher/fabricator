package fabricator

import (
	"fmt"
	"reflect"

	"github.com/bxcodec/faker/v3"
)

type PersistenceHandler[T any] func(instance T) T

type Options[T any] struct {
	PersistenceHandler PersistenceHandler[T]
	Defaults           map[string]any
}

type Factory[T any] struct {
	model              T
	persistenceHandler PersistenceHandler[T]
	defaults           map[string]any
}

// New creates a factory for a struct of type T, receives an optional Options object.
// Options.Defaults is a map of fields names (keys) to default values.
// Options.PersistenceHandler is a function that takes the created struct instance,
// persists it in whatever persistence mechanisms are desired, and returns the resulting struct instance.
func New[T any](model T, opts ...Options[T]) *Factory[T] {
	if reflect.TypeOf(model).Kind() != reflect.Struct {
		panic("unsupported value: model must be a struct")
	}

	var defaults map[string]any
	var handler PersistenceHandler[T]

	if len(opts) > 0 {
		defaults = opts[0].Defaults
		handler = opts[0].PersistenceHandler
	}

	factory := Factory[T]{
		model:              model,
		defaults:           defaults,
		persistenceHandler: handler,
	}

	return &factory
}

// Build creates an instance of the factory's model struct.
// Build takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory Factory[T]) Build(overrides ...map[string]any) T {
	modelType := reflect.TypeOf(factory.model)
	model := reflect.Zero(modelType).Interface().(T)
	if fakerErr := faker.FakeData(&model); fakerErr != nil {
		panic(fmt.Errorf("error generating fake data: %w", fakerErr).Error())
	}
	for key, value := range factory.defaults {
		field := reflect.ValueOf(&model).Elem().FieldByName(key)
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(value))
		}
	}
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			field := reflect.ValueOf(&model).Elem().FieldByName(key)
			if field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(value))
			}
		}
	}
	return model
}

// Batch builds a slice of the factory's model of a given size.
// Batch takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory Factory[T]) Batch(size int, overrides ...map[string]any) []T {
	var batch []T
	for i := 0; i < size; i++ {
		batch = append(batch, factory.Build(overrides...))
	}
	return batch
}

// Create builds an instance of the factory's model struct and persists it using the factory's persistence handler.
// If not persistence handler is defined for the factory, Create will panic.
// Create takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory Factory[T]) Create(overrides ...map[string]any) T {
	if factory.persistenceHandler == nil {
		panic("cannot call .Create on a factory without a persistence handler")
	}
	instance := factory.Build(overrides...)
	return factory.persistenceHandler(instance)
}

// CreateBatch builds a slice of the factory's model of a given size and persists these using the factory's persistence handler.
// CreateBatch takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory Factory[T]) CreateBatch(size int, overrides ...map[string]any) []T {
	if factory.persistenceHandler == nil {
		panic("cannot call .CreateBatch on a factory without a persistence handler")
	}

	var batch []T
	for _, instance := range factory.Batch(size, overrides...) {
		batch = append(batch, factory.persistenceHandler(instance))
	}

	return batch
}
