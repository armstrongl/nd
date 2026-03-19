# Contributing to nd

Thank you for considering contributing to nd! This guide will help you get started.

## Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [git](https://git-scm.com/)
- [golangci-lint](https://golangci-lint.run/) v2: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- [gofumpt](https://github.com/mvdan/gofumpt): `go install mvdan.cc/gofumpt@latest`

## Getting Started

```bash
git clone https://github.com/armstrongl/nd.git
cd nd
go test ./...
go build -o nd .
./nd version
```

## Development Workflow

1. Create a branch from `main`
2. Write tests first (TDD -- red/green/refactor)
3. Implement the feature or fix
4. Run the linter: `golangci-lint run`
5. Format with gofumpt: `gofumpt -w .`
6. Commit with a Conventional Commit message
7. Open a pull request

## Testing

```bash
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

## Adding a New CLI Command

1. Create `cmd/foo.go` with a `func newFooCmd(app *App) *cobra.Command` function
2. Create `cmd/foo_test.go` with tests (write tests first)
3. Register in `cmd/root.go` via `rootCmd.AddCommand(newFooCmd(app))`
4. Follow existing command patterns (see `cmd/deploy.go` as a reference)
5. Add shell completion if the command takes arguments
6. Regenerate command reference: `go run ./cmd/gendocs/`

## Code Style

- **Formatter:** gofumpt (strict superset of gofmt)
- **Linter:** golangci-lint v2 with default configuration
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org/) required

## Commit Messages

Format: `type(scope): description`

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `style`, `ci`, `chore`

**Scopes:** `cli`, `deploy`, `profile`, `source`, `agent`, `config`, `tui`, `docs`, `ci`

Examples:
- `feat(cli): add interactive picker to deploy`
- `fix(deploy): handle broken symlinks on sync`
- `docs: update getting started guide`
- `ci: add coverage upload to CI workflow`

## Pull Requests

- One feature or fix per PR
- Tests required for all code changes
- CI must pass (lint + test + build)
- Reference the issue number if applicable
- Keep PRs focused and reviewable

## Project Structure

See [ARCHITECTURE.md](ARCHITECTURE.md) for a detailed overview of the codebase structure, package responsibilities, and key patterns.
