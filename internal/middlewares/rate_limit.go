package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"github.com/yabeye/addis_verify_backend/pkg/json"
)

// RateLimit uses the custom json.WriteError utility for a consistent response format.
func RateLimit(requests int, window time.Duration, message string) func(next http.Handler) http.Handler {
	return httprate.Limit(
		requests,
		window,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			// Using your reusable helper instead of manual encoding
			json.WriteError(w, http.StatusTooManyRequests, message)
		}),
	)
}
