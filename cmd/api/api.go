package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/yabeye/addis_verify_backend/internal/account"
	repo "github.com/yabeye/addis_verify_backend/internal/database"
	"github.com/yabeye/addis_verify_backend/internal/media"
	"github.com/yabeye/addis_verify_backend/internal/middlewares"
	"github.com/yabeye/addis_verify_backend/internal/users"
	"github.com/yabeye/addis_verify_backend/pkg/auth"
	"github.com/yabeye/addis_verify_backend/pkg/messenger"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/yabeye/addis_verify_backend/docs"
)

type config struct {
	Address string
	DB      struct {
		DSN string
	}
	FallbackURL string
	RedisAddr   string
	JWTSecret   string
	HashPepper  string
}

type application struct {
	config    config
	db        *pgxpool.Pool
	cache     *redis.Client
	logger    *slog.Logger
	messenger messenger.Provider
	auth      auth.TokenManager
}

func (app *application) run(handler http.Handler) error {
	srv := http.Server{
		Addr:         app.config.Address,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  time.Minute,
	}
	return srv.ListenAndServe()
}

// healthCheckHandler godoc
// @Summary      Health Check
// @Description  Checks if the API is up and running
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := time.Now()
	w.Write([]byte(`{"status": "available", "message": "API is live 🚀", "serverTimeStamp": "` + now.Format(time.RFC3339) + `"}`))
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		// ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// PROTECT AGAINST LARGE PAYLOADS: Max 1MB globally.
	r.Use(middlewares.LimitRequestSize(1 * 1024 * 1024))

	// GLOBAL RATE LIMITING: 100 req/min
	r.Use(middlewares.RateLimit(100, 1*time.Minute, "Global rate limit exceeded. Please slow down."))

	// REQUEST TIMEOUT: Don't let connections hang for more than 30s.
	r.Use(middleware.Timeout(30 * time.Second))

	// Compresses large JSON responses (Level 5) to save bandwidth.
	r.Use(middleware.Compress(5))

	// storage
	fileServer := http.FileServer(http.Dir("./store/media"))
	r.Handle("/store/media/*", http.StripPrefix("/store/media/", fileServer))

	// Swagger Route
	r.Get("/swagger/*", httpSwagger.Handler())

	// Health Check (Public)
	r.Get("/health", app.healthCheckHandler)

	// ALL APIS //
	queries := repo.New(app.db)

	accountSvc := account.New(queries)
	accountHandler := account.NewHandler(
		accountSvc,
		app.logger.With("handler", "accounts"),
		app.cache,
		app.messenger,
		app.auth,
		app.config.HashPepper,
	)

	userSvc := users.New(queries)
	mediaSvc := media.NewService("store/media", "http://localhost:8080")
	usersHandler := users.NewHandler(userSvc, mediaSvc, app.logger.With("handler", "users"))

	r.Route("/api/v1", func(r chi.Router) {
		// This mounts Auth (with 10KB limit), Users, and Links from internal/routes
		r.Mount("/", MountRoutes(app, accountHandler, usersHandler))
	})

	return r
}
