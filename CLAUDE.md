# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Test
go test ./...

# Run a single test
go test -run TestName ./path/to/package

# Lint
go vet ./...
```

## Architecture

This is a thin wrapper library (`package httpx`) around Go's standard `net/http`, providing ergonomic HTTP primitives with no external dependencies.

**Core components:**

- **`router.go`** — Central piece. Wraps `http.ServeMux` with a custom `ErrorHandlerFunc` signature (`func(http.ResponseWriter, *http.Request) error`). Routes are registered via `GET/POST/PUT/DELETE/PATCH` methods. `NewGroup` creates sub-routers that share the same underlying mux but prepend a base path and append middlewares. Middleware is applied in reverse-registration order (outermost first at request time). The `router` implements `http.Handler` via `ServeHTTP`.

- **`server.go`** — `NewServer` wraps `*http.Server` with HTTP/1 explicitly enabled via `http.Protocols`.

- **`client.go`** — `NewClient` returns a pre-configured `*http.Client` with sensible timeouts, HTTP/2 forced, and connection pool limits set.

- **`utils/utils.go`** — `package utils` (separate package). Helpers `TextResponse` and `JSONResponse` write typed responses and return any write error to the caller — consistent with the `ErrorHandlerFunc` pattern.

**Key design decision:** Handlers return `error` rather than handling errors internally. Callers supply an `errHandler` adapter (`func(ErrorHandlerFunc) http.HandlerFunc`) to `NewRouter`, which converts errors to HTTP responses — allowing centralized error handling per router instance.
