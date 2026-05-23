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

	appLogger, err := logger.New(logger.Options{Level: cfg.LogLevel, Format: cfg.LogFormat})
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	db, err := sqlite.Open(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("failed to open sqlite database: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(db, "migrations"); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	topicRepo := sqlite.NewTopicRepository(db)
	handlersContainer, err := handlers.NewContainer(topicRepo)
	if err != nil {
		log.Fatalf("failed to initialize handlers: %v", err)
	}

	router := web.NewRouter(handlersContainer)
	server := &http.Server{Addr: cfg.Address(), Handler: router}

	appLogger.Info("http server is starting", "address", cfg.Address())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		appLogger.Error("http server stopped with error", "error", err)
		log.Fatalf("server failed: %v", err)
	}
}
