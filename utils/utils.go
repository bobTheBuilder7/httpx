package utils

import (
	"encoding/json"
	"net/http"
)

func JSONResponse(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}
