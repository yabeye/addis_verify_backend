package middlewares

import (
	"net/http"
)

// LimitRequestSize restricts the size of the request body.
// 'maxSize' is in bytes (e.g., 1024 * 1024 for 1MB).
func LimitRequestSize(maxSize int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// http.MaxBytesReader prevents the body from being read beyond maxSize
			// and automatically closes the connection.
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			next.ServeHTTP(w, r)
		})
	}
}
