package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	repo "github.com/yabeye/addis_verify_backend/internal/database"
	"github.com/yabeye/addis_verify_backend/pkg/auth"
	"github.com/yabeye/addis_verify_backend/pkg/json"
)

// Define a custom type for context keys to prevent collisions
type contextKey string

const UserIDKey contextKey = "user_id"

// AuthMiddleware validates the JWT and checks if the session is still valid in the DB.
func AuthMiddleware(tokenManager auth.TokenManager, db repo.Querier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. SAFE Extraction of the token
			// strings.TrimPrefix is panic-safe even if the header is empty or short
			authHeader := r.Header.Get("Authorization")
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			if tokenString == authHeader || tokenString == "" {
				json.WriteError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
				return
			}

			// 2. Verify Token Signature and Expiry
			claims, err := tokenManager.VerifyToken(tokenString)
			if err != nil {
				json.WriteError(w, http.StatusUnauthorized, "Session expired or invalid token")
				return
			}

			// Ensure it's an access token, not a refresh token
			if claims.Type != "access" {
				json.WriteError(w, http.StatusForbidden, "Only access tokens are permitted")
				return
			}

			// 3. Robust Single Session Check
			if claims.IssuedAt == nil {
				json.WriteError(w, http.StatusUnauthorized, "Invalid token payload: missing iat")
				return
			}

			// Convert string ID from token to pgtype.UUID immediately
			var dbID pgtype.UUID
			if err := dbID.Scan(claims.AccountID); err != nil {
				json.WriteError(w, http.StatusUnauthorized, "Malformed account ID in token")
				return
			}

			// Fetch the account from DB to check TokenValidFrom
			acc, err := db.GetAccountByID(r.Context(), dbID)
			if err != nil {
				// If account doesn't exist anymore
				json.WriteError(w, http.StatusUnauthorized, "User account not found")
				return
			}

			// If the token was issued BEFORE the latest 'token_valid_from' (e.g. after a new login/logout)
			// we invalidate the old token.
			if claims.IssuedAt.Time.Unix() < acc.TokenValidFrom.Time.Unix() {
				json.WriteError(w, http.StatusUnauthorized, "Session has been invalidated by a newer login")
				return
			}

			// 4. Set the scanned UUID into context
			// Storing the object directly saves work for your handlers
			ctx := context.WithValue(r.Context(), UserIDKey, dbID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
