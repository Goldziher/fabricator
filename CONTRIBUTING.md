# Contributing

To contribute code or documentation changes:

1. Fork the upstream repository and clone your fork locally.
2. Install [mise](https://mise.jdx.dev); it installs Go, poly and the other tools from `mise.toml`.
3. Run `mise run setup` to install the tooling and Git hooks.
4. Make focused changes with tests.
5. Run `mise run check`, `mise run test:race`, and `mise run lint`.
6. Open a pull request with a clear explanation of the change.

New behavior should include meaningful tests. The project aims for high coverage, but correctness, failure-path coverage, and maintainable tests matter more than a mechanical 100% target.

## Pre-commit hooks

Install the git hooks with `mise run setup` (or `poly hooks install` directly). On
every commit, poly runs lint, format, and file-safety checks; the commit-msg
hook validates the message. Run all hooks manually with
`poly hooks run pre-commit --all-files`.

## AI governance

The tasks run the latest ai-rulez through npx. Run `mise run governance:generate` after editing `.ai-rulez/`.
Run `mise run governance:check` to validate native OKF sources, the lock, and generated output.

Run `mise run governance:hooks:install` to provision the reviewed validation hooks. Existing machine preferences are preserved.
