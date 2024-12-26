package routes

import (
	"kori/internal/config"
	"kori/internal/handlers"
	"kori/internal/utils/logger"

	"github.com/labstack/echo/v4"
)

func SetupUploadRoutes(api *echo.Group, cfg *config.Config) {
	log := logger.New("upload_routes")

	// Initialize upload handler
	uploadHandler := handlers.NewUploadHandler()

	fileGroup := api.Group("/files")

	fileGroup.POST("/upload", uploadHandler.UploadFile)

	log.Success("Upload routes initialized successfully")
}
