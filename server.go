package httpx

import (
	"net/http"
)

func NewServer(handler http.Handler) *http.Server {
	protocols := new(http.Protocols)

	protocols.SetHTTP1(true)

	server := new(http.Server)

	server.Protocols = protocols
	server.Handler = handler

	return server
}
