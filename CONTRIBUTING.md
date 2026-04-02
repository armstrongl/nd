# Contributing to nd

Thank you for considering contributing to nd! This guide will help you get started.

## Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [git](https://git-scm.com/)
- [golangci-lint](https://golangci-lint.run/) v2: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- [gofumpt](https://github.com/mvdan/gofumpt): `go install mvdan.cc/gofumpt@latest`
- Python 3.11+ (for pre-commit and docs scripts)

## Getting started

```shell
git clone https://github.com/armstrongl/nd.git
cd nd
pip install -r requirements.txt
git config core.hooksPath .githooks
go test ./...
go build -o nd .
./nd version
```

`pip install -r requirements.txt` installs pre-commit and the docs tooling. The first commit after setup will automatically download and install rumdl into a pre-commit-managed environment — no separate rumdl install needed. The `core.hooksPath` line tells git to use `.githooks/pre-commit`, which delegates to pre-commit.

## Development workflow

1. Create a branch from `main`
2. Write tests first (TDD -- red/green/refactor)
3. Implement the feature or fix
4. Run the linter: `golangci-lint run`
5. Format with gofumpt: `gofumpt -w .`
6. Commit with a Conventional Commit message
7. Open a pull request

## Testing

```shell
# Run all unit tests
go test ./...

# Run a specific test
go test ./internal/deploy/ -run TestDeploySymlink -v

# Run with race detection
go test -race ./...

# Run integration tests
go test ./tests/integration/ -v

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Coverage expectations:** 80%+ for business logic packages (`internal/`). Lower coverage is acceptable for CLI (`cmd/`) and TUI (`internal/tui/`) due to interactive code paths.

## Adding a new CLI command

1. Create `cmd/foo.go` with a `func newFooCmd(app *App) *cobra.Command` function
2. Create `cmd/foo_test.go` with tests (write tests first)
3. Register in `cmd/root.go` via `rootCmd.AddCommand(newFooCmd(app))`
4. Follow existing command patterns (see `cmd/deploy.go` as a reference)
5. Add shell completion if the command takes arguments
6. Regenerate command reference: `go run ./cmd/gendocs/`

## Code style

- **Go formatter:** gofumpt (strict superset of gofmt)
- **Go linter:** golangci-lint v2 with default configuration
- **Markdown linter:** [rumdl](https://github.com/rvben/rumdl) — installed automatically by pre-commit
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org/) required

## Commit messages

Format: `type(scope): description`

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `style`, `ci`, `chore`

**Scopes:** `cli`, `deploy`, `profile`, `source`, `agent`, `config`, `tui`, `docs`, `ci`

Examples:

- `feat(cli): add interactive picker to deploy`
- `fix(deploy): handle broken symlinks on sync`
- `docs: update getting started guide`
- `ci: add coverage upload to CI workflow`

## Pull requests

- One feature or fix per PR
- Tests required for all code changes
- CI must pass (lint + test + build)
- Reference the issue number if applicable
- Keep PRs focused and reviewable

## Project structure

See [ARCHITECTURE.md](ARCHITECTURE.md) for a detailed overview of the codebase structure, package responsibilities, and key patterns.
