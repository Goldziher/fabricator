# Contributing

To contribute code or documentation changes:

1. Fork the upstream repository and clone your fork locally.
2. Install Go 1.26, Task, and Homebrew `prek`.
3. Run `task setup` to install Go tooling and Git hooks.
4. Make focused changes with tests.
5. Run `task check`, `task test:race`, and `task lint`.
6. Open a pull request with a clear explanation of the change.

New behavior should include meaningful tests. The project aims for high coverage, but correctness, failure-path coverage, and maintainable tests matter more than a mechanical 100% target.

## Pre-commit hooks

Install the git hooks with `task setup` (or `poly hooks install` directly). On
every commit, poly runs lint, format, and file-safety checks; the commit-msg
hook validates the message. Run all hooks manually with
`poly hooks run pre-commit --all-files`.
