package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yabeye/addis_verify_backend/internal/account"
	"github.com/yabeye/addis_verify_backend/internal/middlewares"
)

// MountRoutes connects the specific sub-handlers for the v1 API.
// Note: Global logic like CORS and basic logging are handled in the main app mount.
func MountRoutes(app *application, accountHandler account.Handler) http.Handler {
	r := chi.NewRouter()

	// 1. AUTHENTICATION ENDPOINTS
	r.Route("/accounts/auth", func(r chi.Router) {

		// Specific Security for Identity Auth:
		// We override the global 1MB limit to 10KB here to prevent
		// "JSON Bomb" attacks on login/signup endpoints.
		r.Use(middlewares.LimitRequestSize(10 * 1024))

		// Stricter rate limiting for Auth (5 req/min) to prevent brute force
		// on phone numbers.
		r.Use(middlewares.RateLimit(5, 1*time.Minute, "Too many login/signup attempts. Try again in a minute."))

		// r.Use(middlewares.Auth(app.auth))
		r.Post("/send-otp", accountHandler.SendOTP)
		r.Post("/verify-otp", accountHandler.VerifyOTP)
	})

	// Future groups (e.g. /identity, /verification) would be mounted here...
	// r.Use(middlewares.Auth(app.auth))

	return r
}
