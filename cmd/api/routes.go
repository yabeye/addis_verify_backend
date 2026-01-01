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

	r.Route("/accounts", func(r chi.Router) {

		// --- PUBLIC AUTH ROUTES ---
		r.Group(func(r chi.Router) {
			// Security: Limit request size to 10KB for auth payloads
			r.Use(middlewares.LimitRequestSize(10 * 1024))
			// Anti-Brute Force: 5 attempts per minute
			r.Use(middlewares.RateLimit(5, 1*time.Minute, "Too many attempts. Try again in a minute."))

			r.Post("/auth/send-otp", accountHandler.SendOTP)
			r.Post("/auth/verify-otp", accountHandler.VerifyOTP)
			r.Post("/auth/refresh", accountHandler.RefreshToken)
		})

		// --- PROTECTED ROUTES ---
		r.Group(func(r chi.Router) {
			// Apply the Auth Middleware to this group only
			r.Use(middlewares.AuthMiddleware(app.auth, app.accountSvc))

			r.Get("/me", accountHandler.GetMe)
			r.Post("/auth/logout", accountHandler.Logout)
			// You can add more protected routes here, like /profile-update
		})
	})

	return r
}
