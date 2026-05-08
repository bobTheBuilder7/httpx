package httpx

import (
	"net/http"
	"strings"

	"github.com/swaggest/openapi-go/openapi31"
)

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

type Middleware func(h ErrorHandlerFunc) ErrorHandlerFunc

type router struct {
	mux         *http.ServeMux
	errHandler  func(h ErrorHandlerFunc) http.HandlerFunc
	basePath    string
	middlewares []Middleware
	Reflector   *openapi31.Reflector
}

func NewRouter(errHandler func(h ErrorHandlerFunc) http.HandlerFunc, middlewares ...Middleware) *router {
	return &router{
		mux:         http.NewServeMux(),
		errHandler:  errHandler,
		basePath:    "",
		middlewares: middlewares,
		Reflector:   openapi31.NewReflector(),
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
		Reflector:   r.Reflector,
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

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func GET[T1, T2 any](r *router, route string, h ErrorHandlerFunc) {
	registerTyped[T1, T2](r, http.MethodGet, route, h)
}

func POST[T1, T2 any](r *router, route string, h ErrorHandlerFunc) {
	registerTyped[T1, T2](r, http.MethodPost, route, h)
}

func PUT[T1, T2 any](r *router, route string, h ErrorHandlerFunc) {
	registerTyped[T1, T2](r, http.MethodPut, route, h)
}

func DELETE[T1, T2 any](r *router, route string, h ErrorHandlerFunc) {
	registerTyped[T1, T2](r, http.MethodDelete, route, h)
}

func PATCH[T1, T2 any](r *router, route string, h ErrorHandlerFunc) {
	registerTyped[T1, T2](r, http.MethodPatch, route, h)
}

func registerTyped[T1, T2 any](r *router, method, route string, h ErrorHandlerFunc) {
	oc, err := r.Reflector.NewOperationContext(method, normalizeRoute(r.basePath+route))
	if err != nil {
		panic(err)
	}
	oc.AddReqStructure(new(T1))
	oc.AddRespStructure(new(T2))
	err = r.Reflector.AddOperation(oc)
	if err != nil {
		panic(err)
	}

	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	r.mux.HandleFunc(method+" "+normalizeRoute(r.basePath+route), r.errHandler(h))
}

func normalizeRoute(route string) string {
	if route == "/" {
		return route + "{$}"
	}

	return strings.TrimSuffix(route, "/")
}
