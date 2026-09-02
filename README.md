<!-- markdownlint-disable MD033 MD041 -->
<div align="center">

<img src="docs/media/fabricator-banner.svg" alt="Fabricator — typed test data for Go" width="820">

**Typed test data factories for Go.**

Say what matters to the test; let the rest be generated.

[![CI](https://img.shields.io/github/actions/workflow/status/Goldziher/fabricator/ci.yaml?style=flat-square&label=ci)](https://github.com/Goldziher/fabricator/actions/workflows/ci.yaml)
[![Go Reference](https://img.shields.io/badge/pkg.go.dev-reference-00ADD8?style=flat-square)](https://pkg.go.dev/github.com/Goldziher/fabricator/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/Goldziher/fabricator?style=flat-square)](https://goreportcard.com/report/github.com/Goldziher/fabricator)
[![Latest Release](https://img.shields.io/github/v/release/Goldziher/fabricator?sort=semver&style=flat-square)](https://github.com/Goldziher/fabricator/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Goldziher/fabricator?style=flat-square)](https://github.com/Goldziher/fabricator/blob/main/go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)

[Docs](https://goldziher.github.io/fabricator/) · [Quickstart](#quickstart) · [Guides](#guides) · [API](https://pkg.go.dev/github.com/Goldziher/fabricator/v2)

</div>

---

Test data written by hand says too much. A test about expired subscriptions needs
one field to be a date in the past, but the struct literal makes you spell out a
name, an email, an ID, and a billing address too. The field that matters is
buried, and every new field on the struct means editing every literal that
constructs it.

Fabricator inverts that. A factory generates a complete value; the test overrides
only what it is about.

```go
expired := factory.Build(fabricator.Override(renewsAt, time.Now().Add(-time.Hour)))
```

Adding a field to `Subscription` does not touch that test.

## Installation

```shell
go get github.com/Goldziher/fabricator/v2
```

Requires Go 1.26 or newer.

## Quickstart

```go
type User struct {
	ID    int
	Name  string
	Email string
	Admin bool
}

var (
	userID    = fabricator.FieldOf[User, int]("ID")
	userName  = fabricator.FieldOf[User, string]("Name")
	userAdmin = fabricator.FieldOf[User, bool]("Admin")
)

factory := fabricator.New(User{},
	fabricator.Field(userID, func(ctx fabricator.BuildContext) int {
		return ctx.Iteration + 1
	}),
)

user := factory.Build()                                    // ID 1, rest generated
admin := factory.Build(fabricator.Override(userAdmin, true))
users := factory.Batch(10)                                 // IDs 1..10
```

`FieldOf[T, V]` checks **when you construct it** that `T` has that field, that it
is exported, and that it accepts a `V`. A typo fails there, with a clear message,
instead of silently doing nothing until an assertion fails somewhere else.

## What you get

**Typed field references.** No `map[string]any`, no stringly-typed configuration
that compiles and then does nothing. `UnsafeFieldOf` is the explicit escape hatch
when a name genuinely is not known statically.

**Generated defaults.** Unconfigured fields are filled by
[go-faker](https://github.com/go-faker/faker), including via `faker:"..."` struct
tags on your model.

**Sequences.**

```go
fabricator.Field(role, fabricator.Sequence("admin", "editor", "viewer"))
factory.Batch(5) // admin, editor, viewer, admin, editor
```

**Factories derived from factories.** `Extend` builds a variant without restating
the base:

```go
base  := fabricator.New(User{}, fabricator.Value(name, "Moishe"), fabricator.Value(role, "user"))
admin := fabricator.Extend(base, fabricator.Value(role, "admin"))
```

**Subfactories** for nested structs, pointers, and slices, with `*With` variants
whose children depend on the parent's build context:

```go
fabricator.Field(favoritePet, fabricator.Subfactory(petFactory))
fabricator.Field(profile, fabricator.PtrSubfactory(profileFactory))
fabricator.Field(pets, fabricator.SliceSubfactory(petFactory, 2))
```

**Lifecycle hooks** at three points — `AfterFaker`, `AfterBuild`, `AfterCreate`:

```go
fabricator.AfterBuild(func(user *User, _ fabricator.BuildContext) error {
	user.Email = strings.ToLower(user.Name) + "@example.com"
	return nil
})
```

**Persistence.** Give a factory a handler and build-and-save is one call:

```go
type PersistenceHandler[T any] interface {
	Save(ctx context.Context, instance T) (T, error)
	SaveMany(ctx context.Context, instances []T) ([]T, error)
}

user := factory.Create(ctx)
users := factory.CreateBatch(ctx, 10)
```

**Errors, not only panics.** `Build`, `Batch`, `Create`, and `CreateBatch` panic
so test bodies stay terse. Each has an `E` twin — `BuildE`, `BatchE`, `CreateE`,
`CreateBatchE` — that returns an error instead.

## Reproducing a failure

Generated data means a test can fail on a value you cannot see. `Seed` pins
generation for the process:

```go
func TestMain(m *testing.M) {
	fabricator.Seed(42)
	os.Exit(m.Run())
}
```

It seeds **both** of faker's sources, including the separate one behind
`faker:"uuid_*"` fields, which would otherwise keep varying. Three limits are
worth knowing, and are spelled out in the
[determinism guide](https://goldziher.github.io/fabricator/guides/determinism/):
do not call `Seed` once builds are running, concurrency still reorders draws, and
`-run`/`-shuffle` change where a test lands in the stream.

## Exact fixtures, and speed

When a test asserts field by field, generated values in the fields it does not
set are noise. `WithoutFaker` starts from the zero value:

```go
factory := fabricator.New(User{},
	fabricator.WithoutFaker[User](),
	fabricator.Value(userName, "Moishe"),
)
factory.Build() // User{Name: "Moishe"} — everything else zero
```

It is deterministic regardless of seed or ordering, and faker's reflective walk
over `T` is what a build actually costs. On a struct of four scalar fields:

| | B/op | allocs/op |
| --- | ---: | ---: |
| `Build` | 2,128 | 45 |
| `Build` with `WithoutFaker` | 64 | 1 |

<sub>`go test -bench .`, Apple M4 Pro, Go 1.27. Allocation counts are quoted
rather than ns/op because they are deterministic; the wall-clock gap on this
shape is roughly 50x, but the exact figure depends on the machine. It gets far
more extreme with collections: faker's default maximum slice and map size is
100, so a struct with one `[]Pet` and one `map[string]string` costs about 1,800
allocations per build, essentially all of it generating elements a test will
never look at.</sub>

## Guides

Full documentation is at **[goldziher.github.io/fabricator](https://goldziher.github.io/fabricator/)**.

- [Introduction](https://goldziher.github.io/fabricator/start/introduction/) — the problem and the shape of the solution
- [Quickstart](https://goldziher.github.io/fabricator/start/quickstart/)
- [Fields and values](https://goldziher.github.io/fabricator/guides/fields/) — references, providers, sequences, precedence
- [Factories from factories](https://goldziher.github.io/fabricator/guides/composition/) — `Extend` and subfactories
- [Lifecycle hooks](https://goldziher.github.io/fabricator/guides/hooks/)
- [Persistence](https://goldziher.github.io/fabricator/guides/persistence/)
- [Determinism](https://goldziher.github.io/fabricator/guides/determinism/)
- [Errors and panics](https://goldziher.github.io/fabricator/reference/errors/) — every message and its cause

## Limits

Factories are for non-pointer struct types. Nested dotted field paths such as
`"Author.Name"` are not supported — use an `AfterBuild` hook to reach into a
nested value.

## Development

```shell
task setup      # tooling and git hooks
task test       # go test ./...
task test:race  # go test -race ./...
task check      # vet, golangci-lint, govulncheck, ai-rulez
task lint       # poly lint and format check
```

Brand assets are generated, not hand-committed:

```shell
python3 scripts/generate_assets.py
```

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
