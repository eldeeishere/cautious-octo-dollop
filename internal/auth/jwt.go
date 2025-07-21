package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateJWT(tokenString string, tokenSecret string) (uuid.UUID, error) {
	calims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, calims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	if !token.Valid {
		return uuid.Nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	userID, err := uuid.Parse(calims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in token: %w", err)
	}
	return userID, nil

}

func GetBearerToken(h http.Header) (string, error) {
	token := h.Get("Authorization")
	if token == "" {
		return "", fmt.Errorf("missing Authorization header")
	}
	if len(token) < 7 || token[:7] != "Bearer " {
		return "", fmt.Errorf("Authorization header must start with 'Bearer '")
	}
	return token[7:], nil
}

func GetApiKey(h http.Header) (string, error) {
	apiKey := h.Get("Authorization")
	if apiKey == "" {
		return "", fmt.Errorf("missing ApiKey header")
	}
	if len(apiKey) < 7 || apiKey[:7] != "ApiKey " {
		return "", fmt.Errorf("X-Polka-Key header must start with 'ApiKey '")
	}
	return apiKey[7:], nil
}
