package utils

import (
	"encoding/json"
	"net/http"
)

func JSONResponse(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
