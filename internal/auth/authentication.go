package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(hashedPassword), err
}

func CheckPasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	})

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return uuid.Nil, fmt.Errorf("This token has expired")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	header, ok := headers["Authorization"]
	if !ok {
		return "", fmt.Errorf("Authorization does not exists in headers")
	}

	token := strings.Replace(header[0], "Bearer", "", -1)
	token = strings.TrimSpace(token)

	if token == "" {
		return token, fmt.Errorf("Bearer token is empty")
	}

	return token, nil
}

func MakeRefreshToken() (string, error) {
	buffer := make([]byte, 32)

	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	header, ok := headers["Authorization"]
	if !ok {
		return "", fmt.Errorf("Authorization does not exists in headers")
	}

	apiKey := strings.Replace(header[0], "ApiKey", "", -1)
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return apiKey, fmt.Errorf("Api key is empty")
	}

	return apiKey, nil
}
