package utils

import (
	"github.com/golang-jwt/jwt/v4"
)

func ValidateRefreshToken(token string, secret string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
