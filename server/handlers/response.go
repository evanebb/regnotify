package handlers

import (
	"encoding/json"
	"net/http"
)

type responseEnvelope struct {
	Status  string `json:"status"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

func writeJSONSuccess(w http.ResponseWriter, statusCode int, v any, message string) {
	writeJSONResponse(w, statusCode, responseEnvelope{
		Status:  "success",
		Data:    v,
		Message: message,
	})
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, responseEnvelope{
		Status:  "error",
		Message: message,
	})
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}
