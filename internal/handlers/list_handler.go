package handlers

import (
	"kori/internal/config"
	"kori/internal/models"
	"kori/internal/templates"
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
// @Router /t/unsubscribe [get]
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

	// Check if contact exists
	if email.Contact == nil {
		return c.String(http.StatusInternalServerError, "Contact not found for this email")
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
	return c.HTML(http.StatusOK, templates.UnsubscribeTemplate(email.Contact.Email, emailID))
}

// HandleEmailResubscribe handles resubscribe requests from email links
// @Summary Resubscribe to email list
// @Description Resubscribe to an email list
// @Accept json
// @Produce json
// @Param email query string true "Resubscribe email"
// @Success 200 {object} map[string]string "Resubscribed successfully"
// @Failure 400 {object} map[string]string "Missing email"
// @Failure 401 {object} map[string]string "Invalid email"
// @Router /t/resubscribe [get]
func (h *TrackingHandler) HandleEmailResubscribe(c echo.Context) error {
	// Extract email from query params
	email := c.QueryParam("id")
	if email == "" {
		return c.String(http.StatusBadRequest, "Missing email")
	}

	// Get the email
	emailModel, err := models.GetEmailByID(email, h.db)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get email")
	}

	// update the contact status
	contact := emailModel.Contact
	contact.Status = models.SubscriberStatusActive
	if err := h.db.Save(contact).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update contact status")
	}

	// Delete the email tracking entry
	if err := h.db.Delete(&models.EmailTracking{}, "email_id = ? and event = ?", emailModel.ID, models.EmailTrackingEventUnsubscribe).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete email tracking entry")
	}

	// Return success page
	return c.HTML(http.StatusOK, templates.ResubscribeTemplate(emailModel.Contact.Email))
}
