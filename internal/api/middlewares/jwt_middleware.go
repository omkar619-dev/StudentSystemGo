package middlewares

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os" // Add this for header parsing if you switch later
	"restapi/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(next http.Handler) http.Handler {
    fmt.Println("-------------JWT MIDDLEWARE------------")

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("--------------- INSIDE JWT MIDDLEWARE")

        // Skip JWT check for login and logout endpoints
        if r.URL.Path == "/execs/login" || r.URL.Path == "/execs/logout" {
            next.ServeHTTP(w, r)
            return
        }

        // Option 1: From cookie (your current approach)
        tokenCookie, err := r.Cookie("Bearer")
        if err != nil {
            http.Error(w, "Authorization cookie missing", http.StatusUnauthorized)
            return
        }
        tokenString := tokenCookie.Value

        // Option 2: From Authorization header (recommended alternative)
        // authHeader := r.Header.Get("Authorization")
        // if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        //     http.Error(w, "Authorization header missing or invalid", http.StatusUnauthorized)
        //     return
        // }
        // tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        jwtSecret := os.Getenv("JWT_SECRET")
        if jwtSecret == "" {
            http.Error(w, "Server configuration error", http.StatusInternalServerError)
            return
        }

        parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return []byte(jwtSecret), nil
        })

        if err != nil {
            if errors.Is(err, jwt.ErrTokenExpired) {
                http.Error(w, "Token expired", http.StatusUnauthorized)
                return
            } else if errors.Is(err, jwt.ErrTokenMalformed) {
                http.Error(w, "Token malformed", http.StatusUnauthorized)
                return
            }
            // Generic error for other issues (e.g., signature invalid)
            log.Printf("JWT parsing error: %v", err) // Log internally, don't expose
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Explicitly check validity
        if !parsedToken.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        } else{
			fmt.Println("Valid token")
		}
		
		fmt.Println("Parsed Token:", parsedToken)

        // Extract and store claims for downstream use
		claims, ok := parsedToken.Claims.(jwt.MapClaims)
        // if ok {
        //     log.Printf("Valid JWT for user: %v, exp: %v", claims["uid"], claims["exp"])
        //     // Store in context
        //     ctx := context.WithValue(r.Context(), "userClaims", claims)
        //     r = r.WithContext(ctx)
        // } else 
		if !ok{
            http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			// log.Println("invalid login token",par)
            return 
        }
		ctx := context.WithValue(r.Context(),utils.ContextKey( "role"),claims["role"])
		ctx = context.WithValue(ctx,utils.ContextKey("expiresAt"),claims["exp"])
		ctx = context.WithValue(ctx,utils.ContextKey("username"),claims["user"])
		ctx = context.WithValue(ctx,utils.ContextKey("userId"),claims["uid"])
		fmt.Println(ctx)
        next.ServeHTTP(w, r.WithContext(ctx))
        fmt.Println("Sent Response from JWT MIDDLEWARE")
    })
}