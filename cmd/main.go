package main

import (
	root "github.com/dinerozz/web-behavior-backend/cmd/root"
	"github.com/dinerozz/web-behavior-backend/config"
	"log"
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	config := config.LoadConfig()
	cmd := root.GetRootCmd(config)

	logger := setupLogger(config.Env)

	logger.Info("starting budget app backend", slog.String("env", config.Env))

	if len(os.Args) == 1 {
		cmd.SetArgs([]string{"serve"})
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
