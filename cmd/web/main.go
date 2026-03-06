package main

import (
	"log"
	"net/http"

	"repetidor/internal/config"
	"repetidor/internal/logger"
	"repetidor/internal/web"
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

	router, err := web.NewRouter()
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
