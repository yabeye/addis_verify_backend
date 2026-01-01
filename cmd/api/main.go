package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/yabeye/addis_verify_backend/internal/env"
	"github.com/yabeye/addis_verify_backend/internal/store"
	"github.com/yabeye/addis_verify_backend/pkg/auth"
	"github.com/yabeye/addis_verify_backend/pkg/messenger"
)

// @title           AddisVerify API
// @version         1.0
// @description     Backend API for AddisVerify identity services.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.addisverify.com/support

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /
func main() {
	// 1. Load Env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 2. Setup Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 3. Config
	cfg := config{
		Address: env.GetString("ADDR", ":8080"),
		DB: struct{ DSN string }{
			DSN: env.GetString("GOOSE_DBSTRING", ""),
		},
		FallbackURL: env.GetString("DEFAULT_FALLBACK_URL", "https://go.dev/"),
		RedisAddr:   env.GetString("REDIS_ADDR", "localhost:6379"),
		JWTSecret:   env.GetString("JWT_SECRET", ""),
		HashPepper:  env.GetString("HASH_PEPPER", "default-dev-pepper-do-not-use-in-prod"),
	}

	// 4. Database Connection
	pool, err := store.NewPostgresPool(cfg.DB.DSN, 10)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	logger.Info("database connection established")

	// 5. Redis Connection
	cache, err := store.NewRedisClient(cfg.RedisAddr, "", 0)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer cache.Close()
	logger.Info("redis connection established")

	jwtManager := auth.NewJWTManager(cfg.JWTSecret)
	smsProvider := messenger.NewMockProvider()

	// 6. Initialize Application
	app := &application{
		config:    cfg,
		db:        pool,
		cache:     cache,
		logger:    logger,
		messenger: smsProvider,
		auth:      jwtManager,
	}

	// 7. Start Server
	logger.Info("starting server", "addr", cfg.Address)
	if err := app.run(app.mount()); err != nil {
		logger.Error("server crashed", "error", err)
		os.Exit(1)
	}
}
