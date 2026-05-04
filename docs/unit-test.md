# Unit Test Guide

## Scope

Unit tests cover pure logic and package-level behavior in:

- `fingerprint/`
- `proxy/`
- `instance/`
- `cloakbrowser/`
- `app/commands/` 中不依赖真实浏览器进程的纯逻辑部分

Tests are co-located with the package under `*_test.go` files.

## Tooling

- Framework: Go standard `testing` package
- Runner: `go test`

Recommended commands:

- `go test -v ./fingerprint/... ./proxy/... ./instance/... ./cloakbrowser/...`
- `go test -v ./...` (runs all package tests; integration-style tests are skipped without env)

## Naming and structure

- File: `*_test.go`
- Test function: `TestXxx`
- Prefer table-driven tests for multiple cases
- Use `t.TempDir()` for filesystem isolation
- Avoid `t.Parallel()` unless shared state is fully isolated

## Mocking rules

Prefer light-weight fakes over heavyweight mocks.

- Store interfaces: use in-memory fakes (see `NewMockStore()` in `instance/` and `proxy/`)
- External browser processes: use dummy binaries or `echo` and inject readiness (`SetReadyFunc`)
- Time-dependent logic: avoid real sleeps when possible; keep timing windows large enough to be stable

Do not mock:

- Core business branching (assert on returned state and errors)
- Public error contracts (e.g., `ErrInstanceNotFound`)

## Test data

Reusable fixtures live under `tests/fixtures/` (instances, proxies and related samples). Prefer loading from these files for consistent scenarios.

## Coverage targets

Targets (not enforced yet):

- Statements: >= 80%
- Branches: >= 75%
- Functions: >= 85%
- Lines: >= 80%

Generate a report with:

```
mkdir -p coverage
go test -coverprofile=coverage/coverage.out -covermode=atomic ./...
go tool cover -html=coverage/coverage.out -o coverage/coverage.html
```
