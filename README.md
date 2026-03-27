# httpx

A thin wrapper around Go's standard `net/http` with ergonomic routing, middleware, and response helpers. No external dependencies.

## Installation

```bash
go get github.com/bobTheBuilder7/httpx
```

## Usage

### Router

Handlers return `error`, and you supply a central `errHandler` to convert errors into HTTP responses.

```go
errHandler := func(h httpx.ErrorHandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := h(w, r); err != nil {
            utils.TextResponse(w, err.Error(), http.StatusInternalServerError)
        }
    }
}

router := httpx.NewRouter(errHandler)

router.GET("/users", listUsers)
router.POST("/users", createUser)
```

### Route Groups

Groups inherit the parent's middlewares and prepend a base path.

```go
api := router.NewGroup("/api/v1", authMiddleware)
api.GET("/users", listUsers)   // → GET /api/v1/users
api.POST("/users", createUser) // → POST /api/v1/users
```

### Middleware

```go
type Middleware func(h httpx.ErrorHandlerFunc) httpx.ErrorHandlerFunc

func logging(next httpx.ErrorHandlerFunc) httpx.ErrorHandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) error {
        log.Println(r.Method, r.URL.Path)
        return next(w, r)
    }
}

router := httpx.NewRouter(errHandler, logging)
```

### Server & Client

```go
// Server with HTTP/1 enabled
server := httpx.NewServer(router)
server.Addr = ":8080"
server.ListenAndServe()

// Client with sensible defaults (HTTP/2, connection pooling, timeouts)
client := httpx.NewClient()
```

### Response Helpers

```go
import "github.com/bobTheBuilder7/httpx/utils"

func handler(w http.ResponseWriter, r *http.Request) error {
    return utils.JSONResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func hello(w http.ResponseWriter, r *http.Request) error {
    return utils.TextResponse(w, "hello", http.StatusOK)
}
```
