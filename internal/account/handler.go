package account

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/yabeye/addis_verify_backend/pkg/auth"
	"github.com/yabeye/addis_verify_backend/pkg/constants"
	"github.com/yabeye/addis_verify_backend/pkg/json"
	"github.com/yabeye/addis_verify_backend/pkg/messenger"
	"github.com/yabeye/addis_verify_backend/pkg/random"
)

// Cache interface abstracts Redis for testability
type Cache interface {
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Pipeline() redis.Pipeliner
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type handler struct {
	service    Service
	logger     *slog.Logger
	cache      Cache
	validate   *validator.Validate
	genOTP     func() (string, error)
	messenger  messenger.Provider
	auth       auth.TokenManager
	hashPepper string
}

type Handler interface {
	SendOTP(w http.ResponseWriter, r *http.Request)
	VerifyOTP(w http.ResponseWriter, r *http.Request)
	RefreshToken(w http.ResponseWriter, r *http.Request)
	GetMe(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}

// NewHandler creates a new account handler with dependencies
func NewHandler(service Service, logger *slog.Logger, cache Cache, messenger messenger.Provider,
	tokenManager auth.TokenManager,
	hashPepper string,
) Handler {
	return &handler{
		service:    service,
		logger:     logger,
		cache:      cache,
		validate:   validator.New(),
		genOTP:     random.GenerateOTP,
		messenger:  messenger,
		auth:       tokenManager,
		hashPepper: hashPepper,
	}
}

func (h *handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOTPRequest

	fmt.Println("SendOTP handler invoked")
	// 1. Decode JSON
	if err := json.Read(r, &req); err != nil {
		h.logger.Error("failed to decode request", "error", err)
		json.WriteError(w, http.StatusBadRequest, constants.ErrInvalidJSON)
		return
	}

	// 2. Validate Inputs
	if err := h.validate.Struct(req); err != nil {
		h.logger.Warn("validation failed", "phone", req.Phone, "error", err)
		json.WriteError(w, http.StatusUnprocessableEntity, constants.ErrInvalidPhoneOrCode)
		return
	}

	ctx := r.Context()
	lockKey := fmt.Sprintf("lock:otp:%s", req.Phone)
	otpKey := fmt.Sprintf("otp:%s", req.Phone)

	// 3. Rate Limit Check
	exists, err := h.cache.Exists(ctx, lockKey).Result()
	if err != nil {
		h.logger.Error("redis error", "error", err)
	}
	if exists > 0 {
		json.WriteError(w, http.StatusTooManyRequests, constants.ErrRateLimit)
		return
	}

	// 4. Generate OTP
	otp, err := h.genOTP()
	if err != nil {
		h.logger.Error("otp generation failed", "error", err)
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// Hash the OTP before saving to Redis
	combined := req.Phone + otp + h.hashPepper
	hash := sha256.Sum256([]byte(combined))
	otpHash := fmt.Sprintf("%x", hash)

	// 5. Store in Cache (Atomic Pipeline)
	pipe := h.cache.Pipeline()
	pipe.Set(ctx, otpKey, otpHash, 5*time.Minute)
	pipe.Set(ctx, lockKey, "locked", 1*time.Minute)

	if _, err := pipe.Exec(ctx); err != nil {
		h.logger.Error("redis pipeline failed", "error", err)
		json.WriteError(w, http.StatusInternalServerError, constants.ErrServiceUnavailable)
		return
	}

	// 6. Send the message professionally
	msg := messenger.Message{
		To:   req.Phone,
		Body: fmt.Sprintf("Your Addis Verify code is: %s. Valid for 5 minutes.", otp),
	}

	// This runs the provider (Mock for now, Twilio later)
	if err := h.messenger.Send(r.Context(), msg); err != nil {
		h.logger.Error("failed to deliver message", "error", err, "phone", req.Phone)
		// We don't necessarily fail the whole request if the SMS provider is slow,
		// but for OTP, it's usually better to return an error.
		json.WriteError(w, http.StatusServiceUnavailable, constants.ErrFailedToSendSMS)
		return
	}

	// 7. Success
	json.Write(w, http.StatusOK, sendOTPResponse{
		Message: constants.MsgOTPSent,
	})
}

// VerifyOTP godoc
// @Summary      Verify OTP and Login
// @Description  Exchanges a 6-digit OTP for an Access and Refresh token pair.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        request  body      verifyOTPRequest  true  "OTP Verification Payload"
// @Success      200      {object}  authSuccessResponse
// @Router       /api/v1/accounts/auth/verify [post]
func (h *handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req verifyOTPRequest

	// 1. Decode and Validate Request
	if err := json.Read(r, &req); err != nil {
		json.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		json.WriteError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	// 2. Retrieve the hashed OTP from Redis
	otpKey := "otp:" + req.Phone
	storedHash, err := h.cache.Get(ctx, otpKey).Result()
	if err != nil {
		h.logger.Warn("OTP expired or not found", "phone", req.Phone)
		json.WriteError(w, http.StatusUnauthorized, constants.ErrInvalidOTP)
		return
	}

	// 3. Verify Hash (OTP + Phone + Server Pepper)
	// This protects against attackers who might see the OTP in transit or access Redis
	combined := req.Phone + req.OTP + h.hashPepper
	inputHash := sha256.Sum256([]byte(combined))
	if fmt.Sprintf("%x", inputHash) != storedHash {
		h.logger.Warn("Invalid OTP attempt", "phone", req.Phone)
		json.WriteError(w, http.StatusUnauthorized, constants.ErrInvalidOTP)
		return
	}

	// 4. Update Database (Atomic Login)
	// This updates 'token_valid_from' to NOW(), invalidating all previous tokens
	dbAccount, err := h.service.UpsertByPhone(ctx, req.Phone)
	if err != nil {
		h.logger.Error("failed to upsert account", "error", err)
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 5. Generate Token Pair (Access + Refresh)
	// We pass dbAccount.TokenValidFrom.Time so the JWT 'iat' matches the DB exactly
	tokenPair, err := h.auth.GenerateTokenPair(dbAccount.ID.String(), dbAccount.TokenValidFrom.Time)
	if err != nil {
		h.logger.Error("failed to generate tokens", "error", err)
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 6. Cleanup Redis
	h.cache.Del(ctx, otpKey)

	// 7. Success Response
	h.logger.Info("user logged in successfully", "account_id", dbAccount.ID)
	json.Write(w, http.StatusOK, authSuccessResponse{
		Message:      "OTP verified successfully",
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		Account:      MapAccountRow(dbAccount),
	})
}

// RefreshToken godoc
// @Summary      Refresh Access Token
// @Description  Rotates the session and provides new tokens. Validates that the refresh token is not older than the last login.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        request  body      object  true  "Refresh Token JSON"
// @Success      200      {object}  authSuccessResponse
// @Router       /api/v1/accounts/auth/refresh [post]
func (h *handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// 1. Define the input structure
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := json.Read(r, &req); err != nil {
		json.WriteError(w, http.StatusBadRequest, constants.ErrInvalidJSON)
		return
	}

	// 2. Verify the Refresh Token (Checks signature and expiry)
	claims, err := h.auth.VerifyToken(req.RefreshToken)
	if err != nil {
		json.WriteError(w, http.StatusUnauthorized, constants.ErrInvalidOrExpiredToken)
		return
	}

	// 3. Security Check: Ensure they aren't trying to use an Access Token to refresh
	if claims.Type != "refresh" {
		json.WriteError(w, http.StatusForbidden, constants.ErrInvalidOrExpiredToken)
		return
	}

	// 4. Convert string ID from token back to pgtype.UUID
	var dbID pgtype.UUID
	if err := dbID.Scan(claims.AccountID); err != nil {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 5. Fetch Account and check if this token is still valid
	acc, err := h.service.GetAccountByID(r.Context(), dbID)
	if err != nil {
		json.WriteError(w, http.StatusUnauthorized, constants.ErrAccountNotFound)
		return
	}

	// Compare JWT IssuedAt with DB ValidFrom (Convert both to Unix for easy comparison)
	if claims.IssuedAt.Time.Unix() < acc.TokenValidFrom.Time.Unix() {
		// "Session invalidated by a newer login"
		json.WriteError(w, http.StatusUnauthorized, constants.ErrInvalidOrExpiredToken)
		return
	}

	// 6. ROTATE: Update token_valid_from in DB to NOW()
	// This makes the CURRENT refresh token unusable for the NEXT request
	newAcc, err := h.service.UpsertByPhone(r.Context(), acc.Phone)
	if err != nil {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 7. Generate NEW pair
	pair, err := h.auth.GenerateTokenPair(newAcc.ID.String(), newAcc.TokenValidFrom.Time)
	if err != nil {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 8. Success Response
	json.Write(w, http.StatusOK, authSuccessResponse{
		Message:      "Tokens rotated successfully",
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		Account:      MapAccountRow(newAcc),
	})
}

// GetMe godoc
// @Summary      Get Current User
// @Description  Returns the profile of the currently authenticated user
// @Tags         accounts
// @Security     BearerAuth
// @Success      200  {object}  AccountDTO
// @Router       /api/v1/accounts/me [get]
func (h *handler) GetMe(w http.ResponseWriter, r *http.Request) {
	// 1. Get the ID stored in the context by the middleware
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	var dbID pgtype.UUID
	dbID.Scan(userIDStr)

	// 2. Fetch fresh data from DB
	acc, err := h.service.GetAccountByID(r.Context(), dbID)
	if err != nil {
		json.WriteError(w, http.StatusNotFound, constants.ErrAccountNotFound)
		return
	}

	// 3. Return mapped DTO
	json.Write(w, http.StatusOK, MapAccountRow(acc))
}

// Logout godoc
// @Summary      Logout User
// @Description  Invalidates all active sessions for the user by rotating the token_valid_from timestamp.
// @Tags         accounts
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Router       /api/v1/accounts/auth/logout [post]
func (h *handler) Logout(w http.ResponseWriter, r *http.Request) {
	// 1. Get the ID stored in the context by the middleware
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 2. Convert string ID back to pgtype.UUID
	var dbID pgtype.UUID
	if err := dbID.Scan(userIDStr); err != nil {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 3. Fetch current account to get the phone number
	acc, err := h.service.GetAccountByID(r.Context(), dbID)
	if err != nil {
		json.WriteError(w, http.StatusNotFound, constants.ErrAccountNotFound)
		return
	}

	// 4. ROTATE: Update token_valid_from to NOW()
	// This effectively "kills" all existing Access and Refresh tokens
	_, err = h.service.UpsertByPhone(r.Context(), acc.Phone)
	if err != nil {
		h.logger.Error("failed to rotate session for logout", "error", err)
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternalServerError)
		return
	}

	// 5. Success
	json.Write(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully. All sessions invalidated.",
	})
}
