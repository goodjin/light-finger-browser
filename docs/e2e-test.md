# E2E Test Guide

## Scope

E2E tests are optional and should cover end-to-end flows that include:

- Real proxy acquisition and release
- Full browser instance lifecycle (start -> health -> stop)
- Browser runtime CDP connectivity

This repository is primarily a Go library. If a full API service is enabled (e.g., via `api-server`), E2E tests can target those endpoints.

## Recommended tooling

- API E2E: `k6`, `newman`, or Go-based tests in a separate `e2e/` package
- Browser E2E: Playwright (only if a UI layer is added)

## Environment setup

Start shared services:

```
docker-compose -f docker-compose.test.yml up -d postgres redis mcp-server
```

`api-server` requires `Dockerfile.api`, which is not present yet.

Provide a real browser binary:

```
export BROWSER_BINARY=/path/to/browser
```

## Example flows to cover

1. Create proxy -> bind -> healthcheck -> release
2. Generate fingerprint -> start instance -> connect CDP -> stop instance
3. Failure path: invalid binary -> expect clean error and no leaked resources

## Execution notes

E2E tests are slow and should be a small set of critical flows. Run before release or after high-risk changes.
