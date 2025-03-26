package main

import (
	"testing"
	"time"

	"github.com/AhmettCelik/web-server/internal/auth"
	"github.com/google/uuid"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		userId      uuid.UUID
		tokenSecret string
		expiresIn   time.Duration
		expectError bool
	}{
		{
			userId:      uuid.New(),
			tokenSecret: "extremely-secret-and-sneak-token!",
			expiresIn:   time.Minute,
			expectError: false,
		},
		{
			userId:      uuid.New(),
			tokenSecret: "extremely-secret-and-sneak-token!",
			expiresIn:   time.Second * 5,
			expectError: true, // Expecting error when token has expired
		},
	}

	for _, c := range cases {
		token, err := auth.MakeJWT(c.userId, c.tokenSecret, c.expiresIn)
		if err != nil {
			t.Errorf("MakeJWT failed: %v", err)
			continue
		}

		if c.expectError {
			time.Sleep(time.Second * 6) // Wait for token to expire
		}

		userID, err := auth.ValidateJWT(token, c.tokenSecret)

		if c.expectError {
			if err == nil {
				t.Error("Expected token to be expired or invalid, but got valid token")
			}
		} else {
			if err != nil {
				t.Errorf("ValidateJWT failed: %v", err)
			}
			if userID != c.userId {
				t.Errorf("Expected userID %v, got %v", c.userId, userID)
			}
		}
	}
}
