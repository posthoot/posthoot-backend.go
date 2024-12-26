package handlers

import (
	"kori/internal/db"
	"kori/internal/models"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"kori/internal/utils/logger"

	"github.com/labstack/echo/v4"
)

type UploadHandler struct {
	log *logger.Logger
}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{
		log: logger.New("upload_handler"),
	}
}

// UploadFile handles file uploads to S3
func (h *UploadHandler) UploadFile(c echo.Context) error {
	storage := GetStorageHandler()
	if storage == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Storage handler not configured",
		})
	}
	// Get file from request
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No file provided",
		})
	}

	// Upload file to S3
	url, err := storage.UploadFile(c.Request().Context(), file, types.ObjectCannedACLPublicRead)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to upload file",
		})
	}

	h.log.Success("File uploaded successfully: %s", url)

	fileModel := &models.File{
		TeamID: c.Get("teamID").(string),
		UserID: c.Get("userID").(string),
		Path:   url[strings.LastIndex(url, "/")+1:],
		Name:   file.Filename,
		Size:   file.Size,
		Type:   file.Header.Get("Content-Type"),
	}

	getDb := db.GetDB()

	// Insert file into database
	err = getDb.Create(fileModel).Error

	if err != nil {
		err := h.log.Error("Failed to insert file into database", err)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to insert file into database",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "File uploaded successfully",
		"file":    fileModel.ID,
	})
}
