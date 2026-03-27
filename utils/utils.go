package utils

import (
	"encoding/json"
	"net/http"
)

func TextResponse(w http.ResponseWriter, text string, status int) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write([]byte(text))
	return err
}

func JSONResponse(w http.ResponseWriter, v any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
