package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"kori/internal/models"
	"kori/internal/utils"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type VerifyResetCodeRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Code     string `json:"code" validate:"required"`
	Password string `json:"new_password" validate:"required,min=8"`
}

// Register handles the registration of a new user by validating input, hashing the password, storing user data, and assigning permissions.
// @Summary Register a new user
// @Description Register a new user with email, password and name details
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} map[string]string "User registered successfully"
// @Failure 400 {object} map[string]string "Validation error or email exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
	}

	// Start a transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start transaction"})
	}

	user := models.User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      models.UserRoleMember, // Default role for new users
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email already exists"})
	}

	// Assign default permissions based on role
	if err := models.AssignDefaultPermissions(tx, &user); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to assign permissions"})
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit transaction"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "User registered successfully"})
}

// Login handles user login by validating credentials, generating a JWT token, and returning it.
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]string "JWT token"
// @Failure 400 {object} map[string]string "Validation error"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	token, err := utils.GenerateJWT(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}
	refreshToken, err := utils.GenerateRefreshToken(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	authtransaction := &models.AuthTransaction{
		UserID: user.ID,
		TeamID: user.TeamID,
		Token:  token,
	}

	if err := h.db.Create(authtransaction).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create auth transaction"})
	}

	return c.JSON(http.StatusOK, map[string]string{"token": token, "refresh_token": refreshToken})
}

// RequestPasswordReset handles the request to reset a user's password by generating a reset code, storing it, and sending an email.
// @Summary Request password reset
// @Description Request a password reset code to be sent via email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Email for password reset"
// @Success 200 {object} map[string]string "Reset code sent if email exists"
// @Failure 400 {object} map[string]string "Validation error"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/password-reset [post]
func (h *AuthHandler) RequestPasswordReset(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return c.JSON(http.StatusOK, map[string]string{"message": "If the email exists, a reset code will be sent"})
	}

	code := generateResetCode()
	reset := models.PasswordReset{
		UserID:    user.ID,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := h.db.Create(&reset).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create reset code"})
	}

	// TODO: Implement email sending functionality
	// sendResetEmail(user.Email, code)

	return c.JSON(http.StatusOK, map[string]string{"message": "If the email exists, a reset code will be sent"})
}

// VerifyResetCode handles the verification of a reset code, updating the user's password, and marking the reset code as used.
// @Summary Verify reset code and set new password
// @Description Verify password reset code and update password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body VerifyResetCodeRequest true "Reset code verification and new password"
// @Success 200 {object} map[string]string "Password reset successful"
// @Failure 400 {object} map[string]string "Invalid or expired reset code"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/password-reset/verify [post]
func (h *AuthHandler) VerifyResetCode(c echo.Context) error {
	var req VerifyResetCodeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid reset code"})
	}

	var reset models.PasswordReset
	if err := h.db.Where("user_id = ? AND code = ? AND used = ? AND expires_at > ?",
		user.ID, req.Code, false, time.Now()).First(&reset).Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid or expired reset code"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
	}

	h.db.Model(&user).Update("password", string(hashedPassword))
	h.db.Model(&reset).Update("used", true)

	return c.JSON(http.StatusOK, map[string]string{"message": "Password reset successfully"})
}

// generateResetCode generates a 6-digit reset code
func generateResetCode() string {
	code := rand.Intn(900000) + 100000 // Generates a number between 100000 and 999999
	return fmt.Sprintf("%06d", code)
}

// ListUsers returns a list of all users (admin only)
// @Summary List all users
// @Description Get a list of all users (requires admin permissions)
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} models.User
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/users [get]
func (h *AuthHandler) ListUsers(c echo.Context) error {
	var users []models.User
	if err := h.db.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch users"})
	}
	return c.JSON(http.StatusOK, users)
}

// GetUser returns details of a specific user (admin only)
// @Summary Get user details
// @Description Get details of a specific user (requires admin permissions)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/users/{id} [get]
func (h *AuthHandler) GetUser(c echo.Context) error {
	id := c.Param("id")
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user"})
	}
	return c.JSON(http.StatusOK, user)
}

// UpdateUser updates a user's details (admin only)
// @Summary Update user details
// @Description Update details of a specific user (requires admin permissions)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body models.User true "Updated user details"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/users/{id} [put]
func (h *AuthHandler) UpdateUser(c echo.Context) error {
	id := c.Param("id")
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user"})
	}

	// Only update allowed fields
	var updateData struct {
		FirstName string          `json:"first_name"`
		LastName  string          `json:"last_name"`
		Role      models.UserRole `json:"role"`
	}

	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Validate role
	if !models.IsValidUserRole(updateData.Role) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid role"})
	}

	user.FirstName = updateData.FirstName
	user.LastName = updateData.LastName
	user.Role = updateData.Role

	if err := h.db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user (admin only)
// @Summary Delete user
// @Description Delete a specific user (requires admin permissions)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string "User deleted successfully"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/users/{id} [delete]
func (h *AuthHandler) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user"})
	}

	if err := h.db.Delete(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// RefreshToken refreshes a user's access token using their refresh token
// @Summary Refresh access token
// @Description Get a new access token using a valid refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_token body string true "Refresh token"
// @Success 200 {object} map[string]string "New access token"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Invalid refresh token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// get refresh token from request
	refreshToken := input.RefreshToken

	// validate refresh token
	_, err := utils.ValidateRefreshToken(refreshToken, os.Getenv("JWT_SECRET"))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid refresh token"})
	}

	// check in db if refresh token is valid
	var authTransaction models.AuthTransaction
	if err := h.db.Where("token = ? AND expires_at > ?", refreshToken, time.Now()).First(&authTransaction).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid refresh token"})
	}

	// get user from claims
	var user models.User
	if err := h.db.First(&user, authTransaction.UserID).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
	}

	// generate new access token
	accessToken, err := utils.GenerateJWT(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate access token"})
	}

	// save new access token to db
	authTransaction.Token = accessToken
	if err := h.db.Save(&authTransaction).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save access token"})
	}

	return c.JSON(http.StatusOK, map[string]string{"token": accessToken})
}

// GetMe returns the current user
// @Summary Get current user
// @Description Get details of the current authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} models.User
// @Router /auth/me [get]
func (h *AuthHandler) GetMe(c echo.Context) error {
	userId := c.Get("userID").(string)

	var user models.User
	if err := h.db.Where("id = ?", userId).First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, user)
}
