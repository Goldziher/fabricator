package fabricator

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
)

type PersistenceHandler[T any] interface {
	Save(instance T) T
	SaveMany(instance []T) []T
}

type Options[T any] struct {
	PersistenceHandler PersistenceHandler[T]
	Defaults           map[string]any
	FakerOptions       []options.OptionFunc
}

type Factory[T any] struct {
	mutex              sync.Mutex
	model              T
	persistenceHandler PersistenceHandler[T]
	defaults           map[string]any
	fakerOpts          []options.OptionFunc
	counter            int
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
	var fakerOpts []options.OptionFunc

	if len(opts) > 0 {
		defaults = opts[0].Defaults
		handler = opts[0].PersistenceHandler
		fakerOpts = opts[0].FakerOptions
	}

	factory := Factory[T]{
		model:              model,
		defaults:           defaults,
		persistenceHandler: handler,
		fakerOpts:          fakerOpts,
	}

	return &factory
}

func setFieldValueIfValid[T any](model *T, iteration int, key string, value any) {
	field := reflect.ValueOf(model).Elem().FieldByName(key)
	if field.IsValid() && field.CanSet() {
		if factoryFunction, isFactoryFunction := value.(func(int, string) interface{}); isFactoryFunction {
			result := factoryFunction(iteration, key)
			field.Set(reflect.ValueOf(result))
		} else {
			field.Set(reflect.ValueOf(value))
		}
	}
}

// ResetCounter resets the factory internal counter to 0.
func (factory *Factory[T]) ResetCounter() {
	factory.counter = 0
}

// GetCounter returns the current value of the factory's internal counter.
func (factory *Factory[T]) GetCounter() int {
	return factory.counter
}

// SetCounter sets the factory's counter to the specified value.
func (factory *Factory[T]) SetCounter(value int) {
	factory.counter = value
}

// Build creates an instance of the factory's model struct.
// Build takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory *Factory[T]) Build(overrides ...map[string]any) T {
	modelType := reflect.TypeOf(factory.model)
	model := reflect.Zero(modelType).Interface().(T)
	if fakerErr := faker.FakeData(&model, factory.fakerOpts...); fakerErr != nil {
		panic(fmt.Errorf("error generating fake data: %w", fakerErr).Error())
	}
	for key, value := range factory.defaults {
		setFieldValueIfValid(&model, factory.counter, key, value)
	}
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			setFieldValueIfValid(&model, factory.counter, key, value)
		}
	}
	factory.mutex.Lock()
	factory.counter++
	factory.mutex.Unlock()
	return model
}

// Batch builds a slice of the factory's model of a given size.
// Batch takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory *Factory[T]) Batch(size int, overrides ...map[string]any) []T {
	var batch []T
	for i := 0; i < size; i++ {
		if len(overrides) > i {
			batch = append(batch, factory.Build(overrides[i]))
		} else {
			batch = append(batch, factory.Build())
		}
	}
	return batch
}

// Create builds an instance of the factory's model struct and persists it using the factory's persistence handler.
// If not persistence handler is defined for the factory, Create will panic.
// Create takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory *Factory[T]) Create(overrides ...map[string]any) T {
	if factory.persistenceHandler == nil {
		panic("cannot call .Create on a factory without a persistence handler")
	}
	instance := factory.Build(overrides...)
	return factory.persistenceHandler.Save(instance)
}

// CreateBatch builds a slice of the factory's model of a given size and persists these using the factory's persistence handler.
// CreateBatch takes an optional map object of overrides.
// Overrides is a map of field names (keys) and values. Overrides take priority over defaults and faker data.
func (factory *Factory[T]) CreateBatch(size int, overrides ...map[string]any) []T {
	if factory.persistenceHandler == nil {
		panic("cannot call .CreateBatch on a factory without a persistence handler")
	}
	batch := factory.Batch(size, overrides...)
	return factory.persistenceHandler.SaveMany(batch)
}
