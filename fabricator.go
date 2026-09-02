package fabricator

import (
	"context"
	"fmt"
	mathrand "math/rand"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
)

// Seed makes faker-generated data reproducible, so a test that fails on
// generated data fails the same way on the next run.
//
// Seed sets the process-wide sources used by github.com/go-faker/faker/v4,
// not per-factory ones, because that is the only seeding faker exposes. Call
// it once from TestMain, before any test runs:
//
//	func TestMain(m *testing.M) {
//		fabricator.Seed(42)
//		os.Exit(m.Run())
//	}
//
// Do not call Seed once builds are under way. Seeding writes faker's package
// level sources without synchronization, so a Seed running concurrently with
// any build is a data race that -race reports, not merely a source of
// surprising values. A single Seed before the builds start is safe: both
// sources installed here are safe for concurrent use, so builds may then run
// in parallel. Their interleaving decides which value each build draws, so
// seeding alone does not make concurrent builds reproducible.
//
// Reproducibility also holds only across runs that generate the same values in
// the same order. Running a subset with -run, or reordering with -shuffle,
// starts a test at a different position in the stream.
//
// Fabricator does not seed by default; without a Seed call faker keeps its own
// default sources.
func Seed(seed int64) {
	// Deliberately math/rand: this seeds test-data generation, where
	// reproducibility is the entire point and unpredictability is not wanted.
	faker.SetRandomSource(faker.NewSafeSource(mathrand.NewSource(seed)))
	// faker draws UUIDs from a second, independent source, so seeding only the
	// first leaves every `faker:"uuid_*"` field varying run to run.
	// G404 is the point: replacing faker's crypto/rand source with a seeded one
	// is what makes uuid-tagged fields reproducible. No security decision reads
	// these bytes.
	faker.SetCryptoSource(&lockedReader{random: mathrand.New(mathrand.NewSource(seed))}) //nolint:gosec
}

// lockedReader makes a math/rand generator usable as faker's crypto source.
// faker reads that source from whichever goroutine is building, and
// *math/rand.Rand is not safe for concurrent use on its own.
type lockedReader struct {
	random *mathrand.Rand
	mu     sync.Mutex
}

func (reader *lockedReader) Read(buffer []byte) (int, error) {
	reader.mu.Lock()
	defer reader.mu.Unlock()

	return reader.random.Read(buffer)
}

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
	skipFaker          bool
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
	factory.defaults = dedupeSetters(factory.defaults)

	return factory
}

// dedupeSetters keeps only the last configuration for each field, in the order
// those last configurations appear.
//
// Without this, configuring a field twice runs both providers and merely lets
// the second value win. That is invisible for a plain value, but a superseded
// provider can have side effects: a Subfactory advances the child factory's
// counter and throws the result away, and a superseded provider that fails
// aborts the build even though its value would never have been used.
func dedupeSetters[T any](setters []fieldSetter[T]) []fieldSetter[T] {
	if len(setters) < 2 {
		return setters
	}

	lastByName := make(map[string]int, len(setters))
	for index, setter := range setters {
		lastByName[setter.name] = index
	}
	if len(lastByName) == len(setters) {
		return setters
	}

	deduped := make([]fieldSetter[T], 0, len(lastByName))
	for index, setter := range setters {
		if lastByName[setter.name] == index {
			deduped = append(deduped, setter)
		}
	}

	return deduped
}

// Extend derives a factory from base, applying opts on top of everything base
// is already configured with, so a variant factory does not have to restate
// the shared configuration.
//
//	base := fabricator.New(User{},
//		fabricator.Value(name, "Moishe"),
//		fabricator.Value(role, "user"),
//	)
//	admin := fabricator.Extend(base, fabricator.Value(role, "admin"))
//
// An option in opts that targets a field base already configures replaces it
// outright: the base's provider for that field does not run, so a superseded
// Subfactory does not build a value that is then discarded. Hooks are additive:
// base's hooks run first, then the ones in opts. A persistence handler in opts
// replaces base's, and WithFaker undoes an inherited WithoutFaker.
//
// The derived factory is independent of base: its counter starts at 0, and
// configuring either factory afterwards does not affect the other. Values
// shared by reference, such as a persistence handler or whatever a provider
// closes over, stay shared.
//
// A fresh counter means an inherited provider that derives a unique value from
// ctx.Iteration no longer produces one across the family: base and every
// factory derived from it each start at iteration 0, so they generate the same
// "unique" email or slug. Where that matters, give the derived factory its own
// provider, or advance its counter with SetCounter.
//
// Extend panics if base is nil.
func Extend[T any](base *Factory[T], opts ...Option[T]) *Factory[T] {
	if base == nil {
		panic("cannot extend a nil factory")
	}

	// Copied at exact length so that appending to the derived factory
	// reallocates instead of writing into base's backing array.
	factory := &Factory[T]{
		persistenceHandler: base.persistenceHandler,
		defaults:           cloneSlice(base.defaults),
		fakerOptions:       cloneSlice(base.fakerOptions),
		afterFaker:         cloneSlice(base.afterFaker),
		afterBuild:         cloneSlice(base.afterBuild),
		afterCreate:        cloneSlice(base.afterCreate),
		skipFaker:          base.skipFaker,
	}
	for _, opt := range opts {
		opt(factory)
	}
	factory.defaults = dedupeSetters(factory.defaults)

	return factory
}

func cloneSlice[S any](src []S) []S {
	if len(src) == 0 {
		return nil
	}

	dst := make([]S, len(src))
	copy(dst, src)

	return dst
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

// WithoutFaker disables faker generation, so builds start from T's zero value
// and only the fields the factory configures are populated.
//
//	factory := fabricator.New(User{},
//		fabricator.WithoutFaker[User](),
//		fabricator.Value(name, "Moishe"),
//	)
//	factory.Build() // User{Name: "Moishe", Email: "", Age: 0}
//
// Use it when random values in unconfigured fields are noise rather than
// coverage, such as a fixture asserted field by field, or one whose unset
// fields must stay empty. It is also substantially faster, since faker's
// reflective walk over T is what dominates a build.
//
// WithFakerOptions has no effect on a factory that skips faker, but AfterFaker
// hooks still run, in the same position, against the zero value.
func WithoutFaker[T any]() Option[T] {
	return func(factory *Factory[T]) {
		factory.skipFaker = true
	}
}

// WithFaker re-enables faker generation. It exists so that a factory derived
// with Extend from a base that used WithoutFaker can turn generation back on,
// which is otherwise the one inherited setting a variant could not vary.
//
// Faker is enabled by default, so this is only worth passing to Extend.
func WithFaker[T any]() Option[T] {
	return func(factory *Factory[T]) {
		factory.skipFaker = false
	}
}

// Sequence returns a provider that cycles through values, one per build,
// selected by the build's iteration.
//
//	fabricator.Field(role, fabricator.Sequence("admin", "editor", "viewer"))
//
//	factory.Batch(5) // admin, editor, viewer, admin, editor
//
// Because the value is chosen by iteration rather than by a counter of its
// own, every Sequence on a factory advances in lockstep, and resetting the
// factory's counter restarts them all.
//
// Sequence panics if no values are given.
func Sequence[V any](values ...V) FieldProvider[V] {
	if len(values) == 0 {
		panic("sequence requires at least one value")
	}

	// Copied so that passing a slice with s... does not leave the provider
	// aliasing the caller's slice, where a later write would change what
	// already-configured factories generate.
	values = cloneSlice(values)

	return func(ctx BuildContext) V {
		// SetCounter accepts negative values, and Go's % keeps the sign.
		index := ctx.Iteration % len(values)
		if index < 0 {
			index += len(values)
		}

		return values[index]
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
	if !factory.skipFaker {
		if fakerErr := faker.FakeData(&model, factory.fakerOptions...); fakerErr != nil {
			return model, 0, fmt.Errorf("error generating fake data: %w", fakerErr)
		}
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
