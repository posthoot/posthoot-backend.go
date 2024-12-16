package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	jwtSecret string
	apiKeys   map[string]APIKeyInfo
}

type APIKeyInfo struct {
	TeamID      string
	Permissions []string
	ExpiresAt   time.Time
}

type Claims struct {
	UserID string   `json:"user_id"`
	TeamID string   `json:"team_id"`
	Email  string   `json:"email"`
	Role   string   `json:"role"`
	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims
}

func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
		apiKeys:   make(map[string]APIKeyInfo),
	}
}

func (m *AuthMiddleware) RegisterAPIKey(key string, info APIKeyInfo) {
	m.apiKeys[key] = info
}

func (m *AuthMiddleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check API Key first
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey != "" {
				return m.validateAPIKey(c, apiKey, next)
			}

			// Check JWT Token
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			return m.validateJWT(c, tokenParts[1], next)
		}
	}
}

func (m *AuthMiddleware) validateAPIKey(c echo.Context, key string, next echo.HandlerFunc) error {
	info, exists := m.apiKeys[key]
	if !exists {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
	}

	// Check expiration
	if !info.ExpiresAt.IsZero() && time.Now().After(info.ExpiresAt) {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key has expired")
	}

	// Set context values
	c.Set("teamID", info.TeamID)
	c.Set("isAPIKey", true)
	c.Set("permissions", info.Permissions)

	return next(c)
}

func (m *AuthMiddleware) validateJWT(c echo.Context, tokenString string, next echo.HandlerFunc) error {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}

	if !token.Valid {
		return echo.NewHTTPError(http.StatusUnauthorized, "Token is not valid")
	}

	// Validate expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return echo.NewHTTPError(http.StatusUnauthorized, "Token has expired")
	}

	// Set context values
	c.Set("userID", claims.UserID)
	c.Set("teamID", claims.TeamID)
	c.Set("email", claims.Email)
	c.Set("role", claims.Role)
	c.Set("scopes", claims.Scopes)
	c.Set("isAPIKey", false)

	return next(c)
}

// Helper functions to get values from context
func GetUserID(c echo.Context) string {
	if id, ok := c.Get("userID").(string); ok {
		return id
	}
	return ""
}

func GetTeamID(c echo.Context) string {
	if id, ok := c.Get("teamID").(string); ok {
		return id
	}
	return ""
}

func GetUserRole(c echo.Context) string {
	if role, ok := c.Get("role").(string); ok {
		return role
	}
	return ""
}

func GetScopes(c echo.Context) []string {
	if scopes, ok := c.Get("scopes").([]string); ok {
		return scopes
	}
	return nil
}

func IsAPIKey(c echo.Context) bool {
	if isAPIKey, ok := c.Get("isAPIKey").(bool); ok {
		return isAPIKey
	}
	return false
}

func HasPermission(c echo.Context, requiredScope string) bool {
	if IsAPIKey(c) {
		if permissions, ok := c.Get("permissions").([]string); ok {
			for _, p := range permissions {
				if p == "ADMIN" || p == requiredScope {
					return true
				}
			}
		}
		return false
	}

	// For JWT tokens, check role and scopes
	role := GetUserRole(c)
	if role == "admin" {
		return true
	}

	scopes := GetScopes(c)
	for _, scope := range scopes {
		if scope == requiredScope {
			return true
		}
	}
	return false
}
