package fabricator

import (
	"context"
	"fmt"
	"reflect"
	"strings"
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

// Hook runs against a generated instance at a lifecycle point.
type Hook[T any] func(*T, BuildContext) error

// Option configures a Factory.
type Option[T any] func(*Factory[T])

// BuildOption configures a single Build, Create, Batch, or CreateBatch call.
type BuildOption[T any] func(*BuildConfig[T])

// PersistenceHandler persists generated instances.
type PersistenceHandler[T any] interface {
	Save(ctx context.Context, instance T) (T, error)
	SaveMany(ctx context.Context, instances []T) ([]T, error)
}

// FieldRef identifies a typed field on T.
type FieldRef[T any, V any] struct {
	name string
}

type fieldSetter[T any] struct {
	name    string
	provide func(BuildContext) any
}

// BuildConfig contains field overrides for a single build call.
type BuildConfig[T any] struct {
	fields []fieldSetter[T]
}

// Factory builds typed test data instances.
type Factory[T any] struct {
	persistenceHandler PersistenceHandler[T]
	defaults           []fieldSetter[T]
	fakerOptions       []options.OptionFunc
	afterFaker         []Hook[T]
	afterBuild         []Hook[T]
	afterCreate        []Hook[T]
	counter            atomic.Int64
}

// New creates a factory for a non-pointer struct of type T.
func New[T any](model T, opts ...Option[T]) *Factory[T] {
	modelType := reflect.TypeOf(model)
	if modelType == nil || modelType.Kind() != reflect.Struct {
		panic("unsupported value: model must be a non-pointer struct")
	}

	factory := &Factory[T]{}
	for _, opt := range opts {
		opt(factory)
	}

	return factory
}

// FieldOf creates a typed reference to a field on T.
func FieldOf[T any, V any](name string) FieldRef[T, V] {
	field, err := lookupField[T](name)
	if err != nil {
		panic(err.Error())
	}

	valueType := reflect.TypeFor[V]()
	if !valueType.AssignableTo(field.Type) {
		panic(fmt.Sprintf("field %q expects %s, got %s", name, field.Type, valueType))
	}

	return FieldRef[T, V]{name: name}
}

// UnsafeFieldOf creates a field reference without validating the field against T.
func UnsafeFieldOf[T any, V any](name string) FieldRef[T, V] {
	return FieldRef[T, V]{name: name}
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

// AfterFaker registers a hook that runs after faker data generation and before field providers.
func AfterFaker[T any](hook Hook[T]) Option[T] {
	return func(factory *Factory[T]) {
		factory.afterFaker = append(factory.afterFaker, hook)
	}
}

// AfterBuild registers a hook that runs after defaults and overrides.
func AfterBuild[T any](hook Hook[T]) Option[T] {
	return func(factory *Factory[T]) {
		factory.afterBuild = append(factory.afterBuild, hook)
	}
}

// AfterCreate registers a hook that runs after persistence.
func AfterCreate[T any](hook Hook[T]) Option[T] {
	return func(factory *Factory[T]) {
		factory.afterCreate = append(factory.afterCreate, hook)
	}
}

// Field configures a typed provider for a struct field.
func Field[T any, V any](field FieldRef[T, V], provider FieldProvider[V]) Option[T] {
	setter := newFieldSetter(field, provider)

	return func(factory *Factory[T]) {
		factory.defaults = append(factory.defaults, setter)
	}
}

// Value configures a static typed value for a struct field.
func Value[T any, V any](field FieldRef[T, V], value V) Option[T] {
	return Field(field, func(BuildContext) V {
		return value
	})
}

// Override configures a typed field value for a single build call.
func Override[T any, V any](field FieldRef[T, V], value V) BuildOption[T] {
	return OverrideField(field, func(BuildContext) V {
		return value
	})
}

// OverrideField configures a typed field provider for a single build call.
func OverrideField[T any, V any](field FieldRef[T, V], provider FieldProvider[V]) BuildOption[T] {
	setter := newFieldSetter(field, provider)

	return func(config *BuildConfig[T]) {
		config.fields = append(config.fields, setter)
	}
}

// Subfactory builds a nested struct value from another factory.
func Subfactory[T any](factory *Factory[T]) FieldProvider[T] {
	mustHaveFactory(factory)

	return func(BuildContext) T {
		return factory.Build()
	}
}

// SubfactoryWith builds a nested struct value using parent-context-derived child overrides.
func SubfactoryWith[T any](factory *Factory[T], overrides func(BuildContext) []BuildOption[T]) FieldProvider[T] {
	mustHaveFactory(factory)

	return func(ctx BuildContext) T {
		return factory.Build(overrides(ctx)...)
	}
}

// PtrSubfactory builds a nested struct pointer from another factory.
func PtrSubfactory[T any](factory *Factory[T]) FieldProvider[*T] {
	mustHaveFactory(factory)

	return func(BuildContext) *T {
		value := factory.Build()
		return &value
	}
}

// PtrSubfactoryWith builds a nested struct pointer using parent-context-derived child overrides.
func PtrSubfactoryWith[T any](factory *Factory[T], overrides func(BuildContext) []BuildOption[T]) FieldProvider[*T] {
	mustHaveFactory(factory)

	return func(ctx BuildContext) *T {
		value := factory.Build(overrides(ctx)...)
		return &value
	}
}

// SliceSubfactory builds a slice of nested structs from another factory.
func SliceSubfactory[T any](factory *Factory[T], size int) FieldProvider[[]T] {
	return SliceSubfactoryWith(factory, func(BuildContext) int {
		return size
	}, nil)
}

// SliceSubfactoryWith builds a dynamic slice of nested structs.
func SliceSubfactoryWith[T any](
	factory *Factory[T],
	size func(BuildContext) int,
	overrides func(BuildContext) []BuildOption[T],
) FieldProvider[[]T] {
	mustHaveFactory(factory)
	if size == nil {
		panic("subfactory size function cannot be nil")
	}

	return func(ctx BuildContext) []T {
		buildOptions := []BuildOption[T](nil)
		if overrides != nil {
			buildOptions = overrides(ctx)
		}

		return factory.Batch(size(ctx), buildOptions...)
	}
}

func newFieldSetter[T any, V any](field FieldRef[T, V], provider FieldProvider[V]) fieldSetter[T] {
	return fieldSetter[T]{
		name: field.name,
		provide: func(ctx BuildContext) any {
			return provider(ctx)
		},
	}
}

func lookupField[T any](name string) (reflect.StructField, error) {
	if strings.Contains(name, ".") {
		return reflect.StructField{}, fmt.Errorf("nested field paths are not supported: %q", name)
	}

	var model T
	modelType := reflect.TypeOf(model)
	if modelType == nil || modelType.Kind() != reflect.Struct {
		return reflect.StructField{}, fmt.Errorf("unsupported value: model must be a non-pointer struct")
	}

	field, ok := modelType.FieldByName(name)
	if !ok {
		return reflect.StructField{}, fmt.Errorf("unknown field %q", name)
	}
	if !field.IsExported() {
		return reflect.StructField{}, fmt.Errorf("field %q cannot be set", name)
	}

	return field, nil
}

func applyFieldSetter[T any](model *T, iteration int, setter fieldSetter[T]) error {
	field := reflect.ValueOf(model).Elem().FieldByName(setter.name)
	if !field.IsValid() {
		return fmt.Errorf("unknown field %q", setter.name)
	}
	if !field.CanSet() {
		return fmt.Errorf("field %q cannot be set", setter.name)
	}

	rawValue := setter.provide(BuildContext{
		Iteration: iteration,
		FieldName: setter.name,
	})
	if rawValue == nil {
		if !isNilAssignable(field.Type()) {
			return fmt.Errorf("field %q expects %s, got nil", setter.name, field.Type())
		}
		field.SetZero()
		return nil
	}

	value := reflect.ValueOf(rawValue)
	if !value.Type().AssignableTo(field.Type()) {
		return fmt.Errorf(
			"field %q expects %s, got %s",
			setter.name,
			field.Type(),
			value.Type(),
		)
	}

	field.Set(value)
	return nil
}

func isNilAssignable(valueType reflect.Type) bool {
	switch valueType.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

func mustHaveFactory[T any](factory *Factory[T]) {
	if factory == nil {
		panic("subfactory cannot use a nil factory")
	}
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err.Error())
	}

	return value
}

func mustSlice[T any](value []T, err error) []T {
	if err != nil {
		panic(err.Error())
	}

	return value
}

// ResetCounter resets the factory internal counter to 0.
//
// ResetCounter is race-safe, but it is not a coordination primitive for concurrent builds.
func (factory *Factory[T]) ResetCounter() {
	factory.counter.Store(0)
}

// GetCounter returns the current value of the factory's internal counter.
func (factory *Factory[T]) GetCounter() int {
	return int(factory.counter.Load())
}

// SetCounter sets the factory's counter to the specified value.
//
// SetCounter is race-safe, but it is not a coordination primitive for concurrent builds.
func (factory *Factory[T]) SetCounter(value int) {
	factory.counter.Store(int64(value))
}

// Build creates an instance of the factory's model struct and panics on errors.
func (factory *Factory[T]) Build(overrides ...BuildOption[T]) T {
	return must(factory.BuildE(overrides...))
}

// BuildE creates an instance of the factory's model struct.
func (factory *Factory[T]) BuildE(overrides ...BuildOption[T]) (T, error) {
	model, _, err := factory.buildE(overrides...)
	return model, err
}

func (factory *Factory[T]) buildE(overrides ...BuildOption[T]) (model T, iteration int, err error) {
	if fakerErr := faker.FakeData(&model, factory.fakerOptions...); fakerErr != nil {
		return model, 0, fmt.Errorf("error generating fake data: %w", fakerErr)
	}

	iteration = int(factory.counter.Add(1) - 1)
	ctx := BuildContext{Iteration: iteration}
	for _, hook := range factory.afterFaker {
		if err := hook(&model, ctx); err != nil {
			return model, iteration, fmt.Errorf("after faker hook failed: %w", err)
		}
	}

	for _, setter := range factory.defaults {
		if err := applyFieldSetter(&model, iteration, setter); err != nil {
			return model, iteration, err
		}
	}

	config := BuildConfig[T]{}
	for _, override := range overrides {
		override(&config)
	}
	for _, setter := range config.fields {
		if err := applyFieldSetter(&model, iteration, setter); err != nil {
			return model, iteration, err
		}
	}

	for _, hook := range factory.afterBuild {
		if err := hook(&model, ctx); err != nil {
			return model, iteration, fmt.Errorf("after build hook failed: %w", err)
		}
	}

	return model, iteration, nil
}

// Batch builds a slice of the factory's model of a given size and panics on errors.
func (factory *Factory[T]) Batch(size int, overrides ...BuildOption[T]) []T {
	return mustSlice(factory.BatchE(size, overrides...))
}

// BatchE builds a slice of the factory's model of a given size.
func (factory *Factory[T]) BatchE(size int, overrides ...BuildOption[T]) ([]T, error) {
	if size < 0 {
		return nil, fmt.Errorf("batch size must be non-negative, got %d", size)
	}

	batch := make([]T, 0, size)
	for range size {
		instance, _, err := factory.buildE(overrides...)
		if err != nil {
			return nil, err
		}
		batch = append(batch, instance)
	}

	return batch, nil
}

// Create builds an instance, persists it, and panics on errors.
func (factory *Factory[T]) Create(ctx context.Context, overrides ...BuildOption[T]) T {
	return must(factory.CreateE(ctx, overrides...))
}

// CreateE builds an instance and persists it using the factory's persistence handler.
func (factory *Factory[T]) CreateE(ctx context.Context, overrides ...BuildOption[T]) (T, error) {
	var zero T
	if factory.persistenceHandler == nil {
		return zero, fmt.Errorf("cannot call .Create on a factory without a persistence handler")
	}

	instance, iteration, err := factory.buildE(overrides...)
	if err != nil {
		return zero, err
	}

	created, err := factory.persistenceHandler.Save(ctx, instance)
	if err != nil {
		return zero, fmt.Errorf("persistence save failed: %w", err)
	}

	hookCtx := BuildContext{Iteration: iteration}
	for _, hook := range factory.afterCreate {
		if err := hook(&created, hookCtx); err != nil {
			return zero, fmt.Errorf("after create hook failed: %w", err)
		}
	}

	return created, nil
}

// CreateBatch builds a batch, persists it, and panics on errors.
func (factory *Factory[T]) CreateBatch(ctx context.Context, size int, overrides ...BuildOption[T]) []T {
	return mustSlice(factory.CreateBatchE(ctx, size, overrides...))
}

// CreateBatchE builds a batch and persists it using the factory's persistence handler.
func (factory *Factory[T]) CreateBatchE(ctx context.Context, size int, overrides ...BuildOption[T]) ([]T, error) {
	if factory.persistenceHandler == nil {
		return nil, fmt.Errorf("cannot call .CreateBatch on a factory without a persistence handler")
	}

	if size < 0 {
		return nil, fmt.Errorf("batch size must be non-negative, got %d", size)
	}

	batch := make([]T, 0, size)
	iterations := make([]int, 0, size)
	for range size {
		instance, iteration, err := factory.buildE(overrides...)
		if err != nil {
			return nil, err
		}
		batch = append(batch, instance)
		iterations = append(iterations, iteration)
	}

	created, err := factory.persistenceHandler.SaveMany(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("persistence save many failed: %w", err)
	}

	for i := range created {
		hookCtx := BuildContext{}
		if i < len(iterations) {
			hookCtx.Iteration = iterations[i]
		}
		for _, hook := range factory.afterCreate {
			if err := hook(&created[i], hookCtx); err != nil {
				return nil, fmt.Errorf("after create hook failed: %w", err)
			}
		}
	}

	return created, nil
}
