package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hey-amanthakur/charrade/apps/backend/internal/config"
	"github.com/hey-amanthakur/charrade/apps/backend/internal/server"
)

func main() {
	cfg := config.Load()
	slog.Info("charrade server starting", "addr", cfg.Addr)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: server.New(cfg),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
