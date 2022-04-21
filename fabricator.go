package fabricator

import (
	"fmt"
	"reflect"

	"github.com/bxcodec/faker/v3"
)

type PersistenceHandler[T any] interface {
	Save(instance T) T
	SaveMany(instances []T) []T
}

type Factory[T any] struct {
	model              T
	persistenceHandler *PersistenceHandler[T]
	fields             map[string]FactoryFunction
	defaults           map[string]any
	settableFields     []string
}

type FactoryFunction func(iteration int) any

// New creates a factory for a model T. Can be extended with a persistenceHandler.
func New[T any](model T) *Factory[T] {
	if modelType := reflect.TypeOf(model); modelType.Kind() == reflect.Struct {
		settableFields := make([]string, 0)
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			if field.IsExported() {
				settableFields = append(settableFields, field.Name)
			}
		}
		factory := Factory[T]{
			model:          model,
			fields:         make(map[string]FactoryFunction),
			defaults:       make(map[string]any),
			settableFields: settableFields,
		}

		return &factory
	}
	panic("unsupported value: model must be a struct")
}

func (factory *Factory[T]) validateFieldName(fieldName string) {
	for _, settableFieldName := range factory.settableFields {
		if fieldName == settableFieldName {
			return
		}
	}
	panic(fmt.Sprintf("%s is either incorrect or does not correlate with a settable field", fieldName))
}

// SetPersistenceHandler sets the factory's persistence handler to the passed in value.
func (factory *Factory[T]) SetPersistenceHandler(persistenceHandler PersistenceHandler[T]) *Factory[T] {
	factory.persistenceHandler = &persistenceHandler
	return factory
}

func (factory *Factory[T]) FieldBuilder(fieldName string, factoryFunction FactoryFunction) *Factory[T] {
	factory.validateFieldName(fieldName)
	factory.fields[fieldName] = factoryFunction
	return factory
}

// Build creates an instance of the factory's model.
func (factory Factory[T]) Build() T {
	modelType := reflect.TypeOf(factory.model)
	model := reflect.Zero(modelType).Interface().(T)
	if fakerErr := faker.FakeData(&model); fakerErr != nil {
		panic(fmt.Sprintf("error generating fake data: %s", fakerErr.Error()))
	}

	return model
}

func (factory Factory[T]) BuildWithOverrides(overrides map[string]any) T {
	model := factory.Build()
	if overrides != nil {
		for key, value := range overrides {
			field := reflect.ValueOf(&model).Elem().FieldByName(key)
			if field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(value))
			}
		}
	}
	return model
}

func (factory Factory[T]) Batch(size int) []T {
	batch := make([]T, size, size)
	for size > 0 {
		batch = append(batch, factory.Build())
	}
	return batch
}

func (factory Factory[T]) BatchWithOverrides(size int, overrides map[string]any) []T {
	batch := make([]T, size, size)
	for size > 0 {
		batch = append(batch, factory.BuildWithOverrides(overrides))
	}
	return batch
}

func (factory Factory[T]) Create() T {
	if factory.persistenceHandler == nil {
		panic("cannot call .Create on a factory without a persistence handler")
	}
	persistenceHandler := *factory.persistenceHandler
	instance := factory.Build()
	return persistenceHandler.Save(instance)
}

func (factory Factory[T]) CreateWithOverrides(overrides map[string]any) T {
	if factory.persistenceHandler == nil {
		panic("cannot call .Create on a factory without a persistence handler")
	}
	persistenceHandler := *factory.persistenceHandler
	instance := factory.BuildWithOverrides(overrides)
	return persistenceHandler.Save(instance)
}

func (factory Factory[T]) CreateBatch(size int) []T {
	if factory.persistenceHandler == nil {
		panic("cannot call .Create on a factory without a persistence handler")
	}
	persistenceHandler := *factory.persistenceHandler
	batch := factory.Batch(size)
	return persistenceHandler.SaveMany(batch)
}

func (factory Factory[T]) CreateBatchWithOverrides(size int, overrides map[string]any) []T {
	if factory.persistenceHandler == nil {
		panic("cannot call .Create on a factory without a persistence handler")
	}
	persistenceHandler := *factory.persistenceHandler
	batch := factory.BatchWithOverrides(size, overrides)
	return persistenceHandler.SaveMany(batch)
}
