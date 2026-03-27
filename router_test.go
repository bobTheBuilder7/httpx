package httpx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bobTheBuilder7/httpx/assert"
)

func testErrHandler(h ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func makeTrackingMiddleware(label string) Middleware {
	return func(next ErrorHandlerFunc) ErrorHandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			fmt.Fprintf(w, "%s:before\n", label)
			err := next(w, r)
			fmt.Fprintf(w, "%s:after\n", label)
			return err
		}
	}
}

func makeTrackingHandler() ErrorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprintln(w, "handler")
		return nil
	}
}

func makeShortCircuitMiddleware(label string, statusCode int) Middleware {
	return func(next ErrorHandlerFunc) ErrorHandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			fmt.Fprintf(w, "%s:short-circuit\n", label)
			w.WriteHeader(statusCode)
			return nil
		}
	}
}

type contextKey string

func makeContextMiddleware(key contextKey, value string) Middleware {
	return func(next ErrorHandlerFunc) ErrorHandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx := context.WithValue(r.Context(), key, value)
			return next(w, r.WithContext(ctx))
		}
	}
}

func TestNoMiddlewares(t *testing.T) {
	r := NewRouter(testErrHandler)
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprint(w, "hello")
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.String(), "hello")
}

func TestWrongHTTPMethod(t *testing.T) {
	r := NewRouter(testErrHandler)
	r.GET("/test", makeTrackingHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusMethodNotAllowed)
}

func TestGroupWithNoExtraMiddlewares(t *testing.T) {
	r := NewRouter(testErrHandler, makeTrackingMiddleware("root"))
	g := r.NewGroup("/api")
	g.GET("/items", makeTrackingHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "root:before\nhandler\nroot:after\n")
}

func TestNewGroupEmptyBasePathPanics(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Equal(t, r.(string), "httpx: basePath must not be empty")
	}()

	rt := NewRouter(testErrHandler)
	rt.NewGroup("")
}

func TestSingleMiddleware(t *testing.T) {
	r := NewRouter(testErrHandler, makeTrackingMiddleware("A"))
	r.GET("/test", makeTrackingHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "A:before\nhandler\nA:after\n")
	assert.Equal(t, rec.Code, http.StatusOK)
}

func TestMultipleMiddlewaresOrder(t *testing.T) {
	r := NewRouter(testErrHandler,
		makeTrackingMiddleware("A"),
		makeTrackingMiddleware("B"),
		makeTrackingMiddleware("C"),
	)
	r.GET("/test", makeTrackingHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "A:before\nB:before\nC:before\nhandler\nC:after\nB:after\nA:after\n")
}

func TestGroupMiddlewaresAfterParent(t *testing.T) {
	r := NewRouter(testErrHandler, makeTrackingMiddleware("parent"))
	g := r.NewGroup("/api", makeTrackingMiddleware("group"))
	g.GET("/items", makeTrackingHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "parent:before\ngroup:before\nhandler\ngroup:after\nparent:after\n")
}

func TestDeeplyNestedGroups(t *testing.T) {
	r := NewRouter(testErrHandler, makeTrackingMiddleware("L0"))
	g1 := r.NewGroup("/v1", makeTrackingMiddleware("L1"))
	g2 := g1.NewGroup("/admin", makeTrackingMiddleware("L2"))
	g3 := g2.NewGroup("/super", makeTrackingMiddleware("L3"))
	g3.GET("/action", makeTrackingHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/super/action", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "L0:before\nL1:before\nL2:before\nL3:before\nhandler\nL3:after\nL2:after\nL1:after\nL0:after\n")
}

func TestAllHTTPMethods(t *testing.T) {
	methods := []struct {
		name     string
		register func(*router, string, ErrorHandlerFunc)
		method   string
	}{
		{"GET", (*router).GET, http.MethodGet},
		{"POST", (*router).POST, http.MethodPost},
		{"PUT", (*router).PUT, http.MethodPut},
		{"DELETE", (*router).DELETE, http.MethodDelete},
		{"PATCH", (*router).PATCH, http.MethodPatch},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			r := NewRouter(testErrHandler, makeTrackingMiddleware("M"))
			m.register(r, "/test", makeTrackingHandler())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(m.method, "/test", nil)
			r.ServeHTTP(rec, req)

			assert.Equal(t, rec.Body.String(), "M:before\nhandler\nM:after\n")
			assert.Equal(t, rec.Code, http.StatusOK)
		})
	}
}

func TestGroupMiddlewareIsolation(t *testing.T) {
	r := NewRouter(testErrHandler, makeTrackingMiddleware("parent"))
	g1 := r.NewGroup("/g1", makeTrackingMiddleware("G1"))
	g2 := r.NewGroup("/g2", makeTrackingMiddleware("G2"))
	g1.GET("/endpoint", makeTrackingHandler())
	g2.GET("/endpoint", makeTrackingHandler())
	r.GET("/root", makeTrackingHandler())

	t.Run("group1_has_parent_and_G1_only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/g1/endpoint", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Body.String(), "parent:before\nG1:before\nhandler\nG1:after\nparent:after\n")
	})

	t.Run("group2_has_parent_and_G2_only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/g2/endpoint", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Body.String(), "parent:before\nG2:before\nhandler\nG2:after\nparent:after\n")
	})

	t.Run("parent_has_parent_middleware_only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/root", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Body.String(), "parent:before\nhandler\nparent:after\n")
	})
}

func TestMiddlewareContextPropagation(t *testing.T) {
	r := NewRouter(testErrHandler,
		makeContextMiddleware("user", "alice"),
		makeContextMiddleware("role", "admin"),
	)

	r.GET("/profile", func(w http.ResponseWriter, r *http.Request) error {
		user := r.Context().Value(contextKey("user")).(string)
		role := r.Context().Value(contextKey("role")).(string)
		fmt.Fprintf(w, "user=%s role=%s", user, role)
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "user=alice role=admin")
	assert.Equal(t, rec.Code, http.StatusOK)
}

func TestMiddlewareShortCircuit(t *testing.T) {
	r := NewRouter(testErrHandler,
		makeTrackingMiddleware("A"),
		makeShortCircuitMiddleware("blocker", http.StatusForbidden),
		makeTrackingMiddleware("C"),
	)
	r.GET("/secret", makeTrackingHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secret", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "A:before\nblocker:short-circuit\nA:after\n")
}

func TestNormalizeRoute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"root_becomes_exact_match", "/", "/{$}"},
		{"no_trailing_slash", "/api/users", "/api/users"},
		{"trailing_slash_trimmed", "/api/users/", "/api/users"},
		{"simple_path", "/api", "/api"},
		{"empty_string", "", ""},
		{"deep_path_trailing_slash", "/a/b/c/", "/a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, normalizeRoute(tt.input), tt.expected)
		})
	}
}

func TestGroupBasePathPrepended(t *testing.T) {
	r := NewRouter(testErrHandler)
	api := r.NewGroup("/api")
	v1 := api.NewGroup("/v1")
	v1.GET("/users", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprint(w, "ok")
		return nil
	})

	t.Run("correct_path_matches", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Code, http.StatusOK)
		assert.Equal(t, rec.Body.String(), "ok")
	})

	t.Run("wrong_paths_return_404", func(t *testing.T) {
		wrongPaths := []string{"/users", "/api/users", "/v1/users", "/api/v1/users/extra"}
		for _, path := range wrongPaths {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			r.ServeHTTP(rec, req)

			assert.Equal(t, rec.Code, http.StatusNotFound)
		}
	})
}

func TestErrorPropagation(t *testing.T) {
	r := NewRouter(testErrHandler, makeTrackingMiddleware("A"))
	r.GET("/fail", func(w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("something broke")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Body.String(), "A:before\nA:after\nsomething broke\n")
}

func TestErrorHandledByMiddleware(t *testing.T) {
	swallower := func(next ErrorHandlerFunc) ErrorHandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			fmt.Fprintln(w, "swallower:before")
			err := next(w, r)
			if err != nil {
				fmt.Fprintln(w, "swallower:caught")
				return nil
			}
			fmt.Fprintln(w, "swallower:after")
			return nil
		}
	}

	r := NewRouter(testErrHandler, swallower)
	r.GET("/fail", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprintln(w, "handler")
		return fmt.Errorf("swallowed error")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.String(), "swallower:before\nhandler\nswallower:caught\n")
}

func TestNewGroupSharesMux(t *testing.T) {
	r := NewRouter(testErrHandler)
	g := r.NewGroup("/api")
	g.GET("/items", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprint(w, "items")
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.String(), "items")
}

func TestMiddlewareSliceIsolation(t *testing.T) {
	r := NewRouter(testErrHandler,
		makeTrackingMiddleware("A"),
		makeTrackingMiddleware("B"),
	)

	g1 := r.NewGroup("/g1", makeTrackingMiddleware("G1"))
	g2 := r.NewGroup("/g2", makeTrackingMiddleware("G2"))

	g1.GET("/test", makeTrackingHandler())
	g2.GET("/test", makeTrackingHandler())

	t.Run("g1_has_A_B_G1", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/g1/test", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Body.String(), "A:before\nB:before\nG1:before\nhandler\nG1:after\nB:after\nA:after\n")
	})

	t.Run("g2_has_A_B_G2", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/g2/test", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Body.String(), "A:before\nB:before\nG2:before\nhandler\nG2:after\nB:after\nA:after\n")
	})
}

func TestRootRouteExactMatch(t *testing.T) {
	r := NewRouter(testErrHandler)
	r.GET("/", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprint(w, "root")
		return nil
	})

	t.Run("root_matches", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Code, http.StatusOK)
		assert.Equal(t, rec.Body.String(), "root")
	})

	t.Run("subpath_does_not_match_root", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/other", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, rec.Code, http.StatusNotFound)
	})
}

func TestMiddlewareSetsResponseHeaders(t *testing.T) {
	addHeader := func(key, value string) Middleware {
		return func(next ErrorHandlerFunc) ErrorHandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) error {
				w.Header().Set(key, value)
				return next(w, r)
			}
		}
	}

	r := NewRouter(testErrHandler,
		addHeader("X-Request-ID", "abc-123"),
		addHeader("X-Powered-By", "httpx"),
	)
	r.GET("/test", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprint(w, "ok")
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Header().Get("X-Request-ID"), "abc-123")
	assert.Equal(t, rec.Header().Get("X-Powered-By"), "httpx")
}
