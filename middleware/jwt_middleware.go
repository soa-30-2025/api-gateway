package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
)

type JWTMiddleware struct {
	Secret               []byte
	AllowUnauthenticated bool // if true, requests without token are forwarded without metadata
	RequireAuthForAll    bool // if true, requests without token => 401
}

func NewJWTMiddlewareFromEnv() *JWTMiddleware {
	secret := os.Getenv("JWT_SECRET")
	return &JWTMiddleware{
		Secret:               []byte(secret),
		AllowUnauthenticated: true,
		RequireAuthForAll:    false,
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r)
		if tokenStr == "" {
			if m.RequireAuthForAll {
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.Secret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		sub, _ := claims["sub"].(string)
		username, _ := claims["username"].(string)
		role, _ := claims["role"].(string)

		md := metadata.Pairs(
			"x-user-id", sub,
			"x-user-username", username,
			"x-user-role", role,
			"x-jwt", tokenStr,
		)
		ctx := metadata.NewIncomingContext(r.Context(), md)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
