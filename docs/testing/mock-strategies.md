# Mock Strategies

This document defines how to mock external dependencies while preserving real risk.

## Dependency list

| Dependency | Test level | Strategy | Reason |
|---|---|---|---|
| Postgres (instances) | Unit | Mock `instance.Store` | Keep unit tests fast and deterministic |
| Postgres (instances) | Integration | Real Postgres via `docker-compose.test.yml` | Validate schema + JSON marshaling |
| Proxy provider | Unit | Fake `ProxyProvider` implementation | Verify error handling and state transitions |
| Proxy provider | Integration | Mock server or fixture-backed fake | Preserve status codes and latency |
| Browser runtime binary | Unit | Dummy script + `SetReadyFunc` | Avoid real process startup |
| Browser runtime binary | Integration/E2E | Real binary | Validate lifecycle and CDP readiness |
| CDP WebSocket | Unit | Stub client with fixed responses | Avoid network flakiness |
| Redis | Integration | Real Redis container | Ensure cache or queue behavior if added |
| File system | Unit | `t.TempDir()` | Avoid user environment dependency |

## Must-preserve behaviors

- Postgres errors and `sql.ErrNoRows` mapping to `ErrInstanceNotFound`
- Proxy healthcheck failures and alert triggers
- Process startup failures (invalid binary) and port release
- CDP connectivity failures (unreachable endpoints, timeouts)

## Acceptable simplifications

- In-memory stores for unit tests (`NewMockStore()` patterns)
- Fixed proxy provider responses for unit tests
- Stubbed CDP responses when validating higher-level logic

## Data strategy

- Use `tests/fixtures/*.json` for repeatable dataset-driven tests
- For integration tests, seed via migrations or explicit setup helpers
- Always clean up containers and temp directories after test runs

## Risks of over-mocking

- Hiding schema or JSON marshaling issues in `instance.PostgresStore`
- Ignoring proxy latency or error responses in health checks
- Missing process cleanup regressions for the browser lifecycle
