package utils

import (
	"os"

	"time"
	"github.com/golang-jwt/jwt/v5"
)

func SignToken(userId int, username, role string) (string, error) {
	// Create the JWT claims, which includes the user ID, username, and role
	jwtSecret := os.Getenv("JWT_SECRET")
	jwtexpiresIn := os.Getenv("JWT_EXPIRES_IN")
	
	claims := jwt.MapClaims{
		"uid": userId,
		"user": username,
		"role": role,

	}
	if jwtexpiresIn != "" {
		expireDuration, err := time.ParseDuration(jwtexpiresIn)
		if err != nil {
			return "", ErrorHandler(err, "invalid JWT_EXPIRES_IN format")
		}
		claims["exp"] = jwt.NewNumericDate(time.Now().Add(expireDuration))
	} else{
		claims["exp"] = jwt.NewNumericDate(time.Now().Add(15 * time.Minute))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", ErrorHandler(err, "failed to sign JWT token")
	}
	return signedToken, nil
}