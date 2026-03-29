package httpx

import (
	"encoding/json"
	"net/http"
	"strings"

	httpxutils "github.com/bobTheBuilder7/httpx/utils"
)

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

type Middleware func(h ErrorHandlerFunc) ErrorHandlerFunc

type router struct {
	mux         *http.ServeMux
	errHandler  func(h ErrorHandlerFunc) http.HandlerFunc
	basePath    string
	middlewares []Middleware
}

func NewRouter(errHandler func(h ErrorHandlerFunc) http.HandlerFunc, middlewares ...Middleware) *router {
	mux := http.NewServeMux()

	return &router{
		mux:         mux,
		errHandler:  errHandler,
		basePath:    "",
		middlewares: middlewares,
	}
}

var _ http.Handler = (*router)(nil)

func (r *router) NewGroup(basePath string, middlewares ...Middleware) *router {
	if basePath == "" {
		panic("httpx: basePath must not be empty")
	}

	mws := make([]Middleware, len(r.middlewares), len(r.middlewares)+len(middlewares))
	copy(mws, r.middlewares)
	mws = append(mws, middlewares...)

	return &router{
		mux:         r.mux,
		errHandler:  r.errHandler,
		basePath:    r.basePath + basePath,
		middlewares: mws,
	}
}

func (r *router) GET(route string, h ErrorHandlerFunc) {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	r.mux.HandleFunc("GET "+normalizeRoute(r.basePath+route), r.errHandler(h))
}

func (r *router) POST(route string, h ErrorHandlerFunc) {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	r.mux.HandleFunc("POST "+normalizeRoute(r.basePath+route), r.errHandler(h))
}

func (r *router) PUT(route string, h ErrorHandlerFunc) {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	r.mux.HandleFunc("PUT "+normalizeRoute(r.basePath+route), r.errHandler(h))
}

func (r *router) DELETE(route string, h ErrorHandlerFunc) {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	r.mux.HandleFunc("DELETE "+normalizeRoute(r.basePath+route), r.errHandler(h))
}

func (r *router) PATCH(route string, h ErrorHandlerFunc) {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	r.mux.HandleFunc("PATCH "+normalizeRoute(r.basePath+route), r.errHandler(h))
}

func DELETE[In any, Out any](r *router, route string, f func(In) (Out, error)) {
	r.DELETE(route, func(w http.ResponseWriter, r *http.Request) error {
		var body In
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return err
		}

		out, err := f(body)
		if err != nil {
			return err
		}

		return httpxutils.JSONResponse(w, out, http.StatusOK)
	})
}

func POST[In any, Out any](r *router, route string, f func(In) (Out, error)) {
	r.POST(route, func(w http.ResponseWriter, r *http.Request) error {
		var body In
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return err
		}

		out, err := f(body)
		if err != nil {
			return err
		}

		return httpxutils.JSONResponse(w, out, http.StatusOK)
	})
}

func PUT[In any, Out any](r *router, route string, f func(In) (Out, error)) {
	r.PUT(route, func(w http.ResponseWriter, r *http.Request) error {
		var body In
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return err
		}

		out, err := f(body)
		if err != nil {
			return err
		}

		return httpxutils.JSONResponse(w, out, http.StatusOK)
	})
}

func PATCH[In any, Out any](r *router, route string, f func(In) (Out, error)) {
	r.PATCH(route, func(w http.ResponseWriter, r *http.Request) error {
		var body In
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return err
		}

		out, err := f(body)
		if err != nil {
			return err
		}

		return httpxutils.JSONResponse(w, out, http.StatusOK)
	})
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func normalizeRoute(route string) string {
	if route == "/" {
		return route + "{$}"
	}

	return strings.TrimSuffix(route, "/")
}
