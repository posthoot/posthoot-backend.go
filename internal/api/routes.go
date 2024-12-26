package api

import (
	"kori/internal/api/middleware"
	"kori/internal/api/registry"
	"kori/internal/routes"
	"net/http"

	_ "kori/docs"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func (s *Server) registerRoutes() {
	s.echo.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	// Health check
	s.echo.GET("/health", s.healthCheck)
	s.echo.GET("/swagger/*", echoSwagger.WrapHandler)
	// API v1 group
	api := s.echo.Group("/api/v1")
	auth := middleware.NewAuthMiddleware(s.config.JWT.Secret)
	api.Use(auth.Middleware())

	// Register CRUD routes for all models
	registry.RegisterCRUDRoutes(api, s.db)

	routes.SetupUploadRoutes(api, s.config)
}
