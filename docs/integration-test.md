# Integration Test Guide

## Scope

Integration tests validate module interactions and external dependencies:

- Database persistence (`instance.PostgresStore`)
- Proxy providers (`proxy.ProxyProvider`)
- Browser process lifecycle (`instance.ProcessManager`)
- Browser runtime client lifecycle (`cloakbrowser.Client`)

Current integration-style tests are co-located with packages (e.g., `instance/process_test.go`) and may be gated by environment variables.

## Environment and services

Start only the services that build from existing Dockerfiles:

```
docker-compose -f docker-compose.test.yml up -d postgres redis mcp-server
```

This starts:

- Postgres (`migrations/` are mounted for schema init)
- Redis
- `mcp-server`

`api-server` currently depends on `Dockerfile.api` (not present). Add that file before enabling it.

Stop and cleanup:

```
docker-compose -f docker-compose.test.yml down -v
```

## Running integration tests

Integration-style tests are currently co-located with packages. Run them directly from the relevant package.

For the existing process lifecycle test, set a real browser binary or packaged browser artifact:

```
export BROWSER_BINARY=/path/to/browser
go test -v ./instance -run TestProcessManager_StartStop
```

## Data setup and cleanup

- Database schema: `migrations/`
- Optional fixtures: `tests/fixtures/*.json`
- Always tear down containers with `make test-down` to reset state

## Assertions

Integration tests must assert on observable behavior:

- Database writes and reads (not just function return values)
- Error contracts on failed dependencies (timeouts, missing rows, invalid binaries)
- Process cleanup (ports released, processes terminated)
