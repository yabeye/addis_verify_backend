package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/yabeye/addis_verify_backend/pkg/auth"
	"github.com/yabeye/addis_verify_backend/pkg/json"
)

// Use a custom type for context keys to avoid collisions
type contextKey string

const UserIDKey contextKey = "user_id"

func Auth(tm auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				json.WriteError(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			// 2. Parse "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				json.WriteError(w, http.StatusUnauthorized, "Invalid authorization format")
				return
			}

			// 3. Validate Token
			userID, err := tm.ValidateToken(parts[1])
			if err != nil {
				json.WriteError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// 4. Inject UserID into context and proceed
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
