package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	tests := []struct {
		name        string
		userID      uuid.UUID
		tokenSecret string
		expiresIn   time.Duration
		wantError   bool
	}{
		{
			name:        "valid token creation",
			userID:      uuid.New(),
			tokenSecret: "test-secret-key",
			expiresIn:   time.Hour,
			wantError:   false,
		},
		{
			name:        "empty secret",
			userID:      uuid.New(),
			tokenSecret: "",
			expiresIn:   time.Hour,
			wantError:   false, // JWT library allows empty secrets
		},
		{
			name:        "zero expiration",
			userID:      uuid.New(),
			tokenSecret: "test-secret-key",
			expiresIn:   0,
			wantError:   false,
		},
		{
			name:        "negative expiration",
			userID:      uuid.New(),
			tokenSecret: "test-secret-key",
			expiresIn:   -time.Hour,
			wantError:   false,
		},
		{
			name:        "nil UUID",
			userID:      uuid.Nil,
			tokenSecret: "test-secret-key",
			expiresIn:   time.Hour,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := MakeJWT(tt.userID, tt.tokenSecret, tt.expiresIn)

			if tt.wantError {
				if err == nil {
					t.Errorf("MakeJWT() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("MakeJWT() unexpected error: %v", err)
				return
			}

			if token == "" {
				t.Errorf("MakeJWT() returned empty token")
			}

			// Verify token has correct structure (JWT has 3 parts separated by dots)
			dotCount := 0
			for _, char := range token {
				if char == '.' {
					dotCount++
				}
			}
			if dotCount != 2 {
				t.Errorf("MakeJWT() returned invalid JWT format, expected 2 dots, got %d", dotCount)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	validUserID := uuid.New()
	validSecret := "test-secret-key"
	validToken, _ := MakeJWT(validUserID, validSecret, time.Hour)

	// Create an expired token
	expiredToken, _ := MakeJWT(validUserID, validSecret, -time.Hour)

	// Create a token with different secret
	differentSecretToken, _ := MakeJWT(validUserID, "different-secret", time.Hour)

	tests := []struct {
		name        string
		tokenString string
		tokenSecret string
		wantUserID  uuid.UUID
		wantError   bool
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			tokenSecret: validSecret,
			wantUserID:  validUserID,
			wantError:   false,
		},
		{
			name:        "expired token",
			tokenString: expiredToken,
			tokenSecret: validSecret,
			wantUserID:  uuid.Nil,
			wantError:   true,
		},
		{
			name:        "wrong secret",
			tokenString: differentSecretToken,
			tokenSecret: validSecret,
			wantUserID:  uuid.Nil,
			wantError:   true,
		},
		{
			name:        "malformed token",
			tokenString: "invalid.token.format",
			tokenSecret: validSecret,
			wantUserID:  uuid.Nil,
			wantError:   true,
		},
		{
			name:        "empty token",
			tokenString: "",
			tokenSecret: validSecret,
			wantUserID:  uuid.Nil,
			wantError:   true,
		},
		{
			name:        "empty secret",
			tokenString: validToken,
			tokenSecret: "",
			wantUserID:  uuid.Nil,
			wantError:   true,
		},
		{
			name:        "completely invalid token",
			tokenString: "not-a-jwt-token",
			tokenSecret: validSecret,
			wantUserID:  uuid.Nil,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := ValidateJWT(tt.tokenString, tt.tokenSecret)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateJWT() expected error, got nil")
				}
				if userID != uuid.Nil {
					t.Errorf("ValidateJWT() expected uuid.Nil on error, got %v", userID)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateJWT() unexpected error: %v", err)
				return
			}

			if userID != tt.wantUserID {
				t.Errorf("ValidateJWT() got userID = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

func TestMakeJWTAndValidateJWT_Integration(t *testing.T) {
	userID := uuid.New()
	secret := "integration-test-secret"
	expiresIn := time.Hour

	// Create a token
	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() failed: %v", err)
	}

	// Validate the token
	validatedUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT() failed: %v", err)
	}

	// Check if the user ID matches
	if validatedUserID != userID {
		t.Errorf("Integration test failed: expected userID %v, got %v", userID, validatedUserID)
	}
}

func TestValidateJWT_DifferentSigningMethods(t *testing.T) {
	secret := "test-secret"

	// Use a pre-made token string with RS256 algorithm (which should be rejected)
	// This token has RS256 in the header which our validation should reject
	tokenString := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiJ0ZXN0LXVzZXItaWQiLCJleHAiOjk5OTk5OTk5OTksImlhdCI6MTYwMDAwMDAwMH0.invalid"

	_, err := ValidateJWT(tokenString, secret)
	if err == nil {
		t.Errorf("ValidateJWT() should reject RS256 tokens, got nil error")
	}
}

func TestJWT_ClaimsValidation(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	// Test token creation and validation with specific claims
	token, err := MakeJWT(userID, secret, time.Minute*30)
	if err != nil {
		t.Fatalf("MakeJWT() failed: %v", err)
	}

	// Parse the token to verify claims
	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !parsedToken.Valid {
		t.Errorf("Token should be valid")
	}

	// Verify issuer
	if claims.Issuer != "chirpy" {
		t.Errorf("Expected issuer 'chirpy', got '%s'", claims.Issuer)
	}

	// Verify subject
	if claims.Subject != userID.String() {
		t.Errorf("Expected subject '%s', got '%s'", userID.String(), claims.Subject)
	}

	// Verify expiration is in the future
	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Errorf("Token should not be expired yet")
	}

	// Verify issued at is in the past (within last minute)
	if claims.IssuedAt.Time.After(time.Now()) {
		t.Errorf("IssuedAt should be in the past")
	}
}

func TestValidateJWT_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		tokenSecret string
		wantError   bool
	}{
		{
			name:        "token with only header",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			tokenSecret: "secret",
			wantError:   true,
		},
		{
			name:        "token with header and payload but no signature",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
			tokenSecret: "secret",
			wantError:   true,
		},
		{
			name:        "token with invalid base64",
			tokenString: "invalid.base64.encoding!!!",
			tokenSecret: "secret",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateJWT(tt.tokenString, tt.tokenSecret)
			if tt.wantError && err == nil {
				t.Errorf("ValidateJWT() expected error for %s, got nil", tt.name)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateJWT() unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMakeJWT(b *testing.B) {
	userID := uuid.New()
	secret := "benchmark-secret"
	expiresIn := time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MakeJWT(userID, secret, expiresIn)
		if err != nil {
			b.Fatalf("MakeJWT() failed: %v", err)
		}
	}
}

func BenchmarkValidateJWT(b *testing.B) {
	userID := uuid.New()
	secret := "benchmark-secret"
	token, _ := MakeJWT(userID, secret, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidateJWT(token, secret)
		if err != nil {
			b.Fatalf("ValidateJWT() failed: %v", err)
		}
	}
}
