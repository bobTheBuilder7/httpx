package utils

import (
	"encoding/json"
	"net/http"
)

func TextResponse(w http.ResponseWriter, text string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte(text))
	return err
}

func JSONResponse(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}
