package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type Server struct {
	App *fiber.App
	Cfg *config.Config
}

func New(cfg *config.Config) *Server {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
		AppName:      "PKL Backend API v2",
		BodyLimit:    10 * 1024 * 1024, // 20 MB limit
	})

	// Global middlewares
	app.Use(recover.New())
	app.Use(requestid.New(requestid.Config{
		Header: "X-Request-ID",
	}))
	app.Use(middleware.NewCORS(cfg))
	app.Use(middleware.NewLogger())

	// Basic health check route
	app.Get("/health", func(c *fiber.Ctx) error {
		return response.Single(c, fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"env":       cfg.AppEnv,
		}, "PKL Backend is healthy")
	})

	return &Server{
		App: app,
		Cfg: cfg,
	}
}

func (s *Server) Run() error {
	addr := fmt.Sprintf("0.0.0.0:%s", s.Cfg.Port)

	// Channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Server is listening", "address", addr, "env", s.Cfg.AppEnv)
		if err := s.App.Listen(addr); err != nil {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		slog.Info("Shutting down server gracefully...", "signal", sig.String())
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.App.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	slog.Info("Server stopped cleanly.")
	return nil
}
