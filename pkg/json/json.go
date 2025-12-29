package json

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents the standard error format for AddisVerify.
// @Description Standard error response body
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}

// Write encodes data as JSON.
func Write(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError sends a structured error response.
func WriteError(w http.ResponseWriter, code int, msg string) {
	// Use the struct here instead of a map
	Write(w, code, ErrorResponse{Error: msg})
}

// Decode decodes a JSON request body into dst.
func Decode(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

// Redirect sends an HTTP redirect to the given URL.
func Redirect(w http.ResponseWriter, r *http.Request, url string, code int) {
	http.Redirect(w, r, url, code)
}
