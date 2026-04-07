package main

import (
	"log"
	"net/http"

	"repetidor/internal/config"
	"repetidor/internal/logger"
	"repetidor/internal/sqlite"
	"repetidor/internal/web"
	"repetidor/internal/web/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	appLogger, err := logger.New(logger.Options{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	appLogger.Info(
		"starting repetidor",
		"app_env", cfg.AppEnv,
		"http_host", cfg.HTTPHost,
		"http_port", cfg.HTTPPort,
		"sqlite_path", cfg.SQLitePath,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
	)

	db, err := sqlite.Open(cfg.SQLitePath)
	if err != nil {
		appLogger.Error("failed to open sqlite database", "error", err)
		log.Fatalf("failed to open sqlite database: %v", err)
	}
	defer db.Close()

	appLogger.Info("sqlite database opened", "sqlite_path", cfg.SQLitePath)

	handlersContainer, err := handlers.NewContainer()
	if err != nil {
		log.Fatalf("failed to initialize handlers: %v", err)
	}

	router := web.NewRouter(handlersContainer)
	if err != nil {
		log.Fatalf("failed to initialize router: %v", err)
	}

	server := &http.Server{
		Addr:    cfg.Address(),
		Handler: router,
	}

	appLogger.Info("http server is starting", "address", cfg.Address())

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		appLogger.Error("http server stopped with error", "error", err)
		log.Fatalf("server failed: %v", err)
	}
}
