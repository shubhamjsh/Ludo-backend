package middleware

import (
	"context"
	"net/http"
	"strings"

	"Ludo/utils"
)

// AuthMiddleware validates JWT tokens and adds user info to request context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.SendError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.SendError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		token := parts[1]

		// Validate token
		claims, err := utils.ValidateToken(token)
		if err != nil {
			if err == utils.ErrExpiredToken {
				utils.SendError(w, http.StatusUnauthorized, "Token has expired")
				return
			}
			utils.SendError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		ctx = context.WithValue(ctx, "isGuest", claims.IsGuest)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

