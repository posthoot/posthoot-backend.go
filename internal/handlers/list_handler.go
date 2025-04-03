package handlers

import (
	"kori/internal/config"
	"kori/internal/models"
	"net/http"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

// HandleEmailUnsubscribe handles unsubscribe requests from email links
// @Summary Unsubscribe from email list
// @Description Unsubscribe from an email list
// @Accept json
// @Produce json
// @Param token query string true "Unsubscribe token"
// @Success 200 {object} map[string]string "Unsubscribed successfully"
// @Failure 400 {object} map[string]string "Missing token"
// @Failure 401 {object} map[string]string "Invalid token"
// @Router /track/unsubscribe [get]

func (h *TrackingHandler) HandleEmailUnsubscribe(c echo.Context) error {
	// Extract token from query params
	token := c.QueryParam("token")
	if token == "" {
		return c.String(http.StatusBadRequest, "Missing token")
	}

	// Parse JWT token
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GetConfig().JWT.Secret), nil
	})

	if err != nil {
		return c.String(http.StatusUnauthorized, "Invalid token")
	}

	// Extract email ID and recipient email from claims
	emailID, ok := claims["mailId"].(string)
	if !ok {
		return c.String(http.StatusBadRequest, "Invalid token claims - missing email ID")
	}

	// Get the email
	email, err := models.GetEmailByID(emailID, h.db)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get email")
	}

	// update the contact status
	contact := email.Contact
	contact.Status = models.SubscriberStatusUnsubscribed
	if err := h.db.Save(contact).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update contact status")
	}

	// Create tracking entry for the unsubscribe event
	_, err = h.createTrackingEntry(c, emailID, models.EmailTrackingEventUnsubscribe, "")
	if err != nil {
		// Log error but don't fail the request
		trackingLog.Error("Failed to create unsubscribe tracking entry", err)
	}

	// Return success page
	return c.HTML(http.StatusOK, "<h1>Successfully Unsubscribed</h1><p>You have been removed from our mailing list.</p>")
}
