package utils

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/argon2"
	"strings"
	"fmt"
	"crypto/rand"
)

func VerifyPassword(password, encodedHash string) error {
	parts := strings.Split(encodedHash, ".")
	if len(parts) != 2 {
		return ErrorHandler(errors.New("invalid password format"), "internal server error")
		// http.Error(w, "invalid password format", http.StatusInternalServerError)
		// return true
	}

	saltBase64 := parts[0]
	hashedPasswordBase64 := parts[1]

	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return ErrorHandler(err, "internal server error")
	}
	hashedPassword, err := base64.StdEncoding.DecodeString(hashedPasswordBase64)
	if err != nil {
		return ErrorHandler(err, "internal server error")
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 3, 32)

	if len(hash) != len(hashedPassword) {
		return ErrorHandler(errors.New("invalid password, hash length mismatch"), "password does not match")
	}
	if subtle.ConstantTimeCompare(hash, hashedPassword) == 1 {
		return nil
	}

	// do nothing, password is correct
	return ErrorHandler(errors.New("incorrect password"), "incorrect password")

}
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrorHandler(errors.New("Password is required"), "please enter a password for the exec")
	}
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", ErrorHandler(errors.New("failed to generate salt"), "failed to generate salt for password hashing")
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 3, 32)
	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	hashBase64 := base64.StdEncoding.EncodeToString(hash)
	encodedHash := fmt.Sprintf("%s.%s", saltBase64, hashBase64)
	return encodedHash, nil
}
