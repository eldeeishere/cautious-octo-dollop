package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bcryptHash), nil
}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32) // 256 bits
	n, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	if n != 32 {
		return "", fmt.Errorf("expected 32 bytes, got %d", n)
	}
	encodedStr := hex.EncodeToString(key)
	return encodedStr, nil
}
