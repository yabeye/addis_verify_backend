package json

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents the standard error format for AddisVerify.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}

// Write encodes data as JSON and sends it to the client.
func Write(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError sends a structured error response.
func WriteError(w http.ResponseWriter, code int, msg string) {
	Write(w, code, ErrorResponse{Error: msg})
}

// Read decodes a JSON request body into dst.
// We renamed this from Decode to Read to match your handler's "req.Bind" call.
func Read(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)

	// Optional: This prevents clients from sending random fields
	// that aren't defined in your DTO.
	decoder.DisallowUnknownFields()

	defer r.Body.Close() // Good practice to close the body after reading
	return decoder.Decode(dst)
}

// Redirect sends an HTTP redirect to the given URL.
func Redirect(w http.ResponseWriter, r *http.Request, url string, code int) {
	http.Redirect(w, r, url, code)
}
