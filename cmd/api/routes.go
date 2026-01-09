package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yabeye/addis_verify_backend/internal/account"
	repo "github.com/yabeye/addis_verify_backend/internal/database"
	"github.com/yabeye/addis_verify_backend/internal/middlewares"
	"github.com/yabeye/addis_verify_backend/internal/users"
)

// MountRoutes connects the specific sub-handlers for the v1 API.
func MountRoutes(app *application, accountHandler account.Handler, userHandler users.Handler) http.Handler {
	r := chi.NewRouter()
	queries := repo.New(app.db)

	// --- ACCOUNT ROUTES ---
	r.Route("/accounts", func(r chi.Router) {
		// Public Auth Routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.LimitRequestSize(10 * 1024))
			r.Use(middlewares.RateLimit(5, 1*time.Minute, "Too many attempts."))

			r.Post("/auth/send-otp", accountHandler.SendOTP)
			r.Post("/auth/verify-otp", accountHandler.VerifyOTP)
			r.Post("/auth/refresh", accountHandler.RefreshToken)
		})

		// Protected Account Routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware(app.auth, queries))
			r.Get("/me", accountHandler.GetMe)
			r.Post("/auth/logout", accountHandler.Logout)
		})
	})

	// --- USER & PROFILE ROUTES ---
	r.Route("/users", func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware(app.auth, queries))

		r.Get("/me", userHandler.GetMe)
		r.Put("/profile", userHandler.UpdateProfile)

		// Step 1: Frontend gets a "Ticket" (the upload URL)
		r.Get("/profile/upload-url", userHandler.GetUploadURL)
	})

	// --- MEDIA & STORAGE (Pattern 1) ---

	// Step 2: The "Door B" - This is where the actual file bits are sent.
	// We put this outside /users if you want it to be a dedicated media endpoint.
	r.Route("/media", func(r chi.Router) {
		// We protect the upload because it writes to our disk/storage
		r.Use(middlewares.AuthMiddleware(app.auth, queries))

		// The path becomes: /api/v1/media/upload/{userID}/{fileName}
		r.Put("/upload/{userID}/{fileName}", userHandler.HandleBinaryUpload)
	})

	// Step 3: Serve static files so the frontend can display images.
	// This maps http://localhost:8080/store/media/image.jpg to ./store/media/image.jpg
	fileDir := "./store/media"
	fileServer := http.FileServer(http.Dir(fileDir))

	// We use Handle because FileServer returns an http.Handler
	r.Handle("/store/media/*", http.StripPrefix("/store/media/", fileServer))

	return r
}
