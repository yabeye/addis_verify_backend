package middlewares

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yabeye/addis_verify_backend/internal/account" // Import your service interface
	"github.com/yabeye/addis_verify_backend/pkg/auth"         // Import your token manager
	"github.com/yabeye/addis_verify_backend/pkg/json"
)

// AuthMiddleware now takes its dependencies as arguments
func AuthMiddleware(tokenManager auth.TokenManager, svc account.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
				json.WriteError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
				return
			}

			tokenString := authHeader[7:]

			// 2. Verify Token
			claims, err := tokenManager.VerifyToken(tokenString)
			if err != nil {
				json.WriteError(w, http.StatusUnauthorized, "Session expired or invalid token")
				return
			}

			if claims.Type != "access" {
				json.WriteError(w, http.StatusForbidden, "Refresh tokens cannot be used for this endpoint")
				return
			}

			// 3. Single Session Check
			var dbID pgtype.UUID
			dbID.Scan(claims.AccountID)

			acc, err := svc.GetAccountByID(r.Context(), dbID)
			if err != nil || claims.IssuedAt.Time.Unix() < acc.TokenValidFrom.Time.Unix() {
				json.WriteError(w, http.StatusUnauthorized, "Session invalidated by a newer login")
				return
			}

			// 4. Success: Set context
			ctx := context.WithValue(r.Context(), "user_id", claims.AccountID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
