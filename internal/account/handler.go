package account

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
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
	service   Service
	logger    *slog.Logger
	cache     Cache
	validate  *validator.Validate
	genOTP    func() (string, error)
	messenger messenger.Provider
	auth      auth.TokenManager
}

type Handler interface {
	SendOTP(w http.ResponseWriter, r *http.Request)
	VerifyOTP(w http.ResponseWriter, r *http.Request)
}

// NewHandler creates a new account handler with dependencies
func NewHandler(service Service, logger *slog.Logger, cache Cache, messenger messenger.Provider,
	tokenManager auth.TokenManager,
) Handler {
	return &handler{
		service:   service,
		logger:    logger,
		cache:     cache,
		validate:  validator.New(),
		genOTP:    random.GenerateOTP,
		messenger: messenger,
		auth:      tokenManager,
	}
}

// SendOTP godoc
// @Summary      Send OTP to phone
// @Description  Generates a 6-digit OTP, hashes it, stores it in Redis with a 1-minute rate limit lock, and sends it via SMS.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        request  body      sendOTPRequest  true  "Phone number in E.164 format (must start with +)"
// @Success      200      {object}  sendOTPResponse
// @Failure      400      {object}  map[string]string "Invalid JSON"
// @Failure      422      {object}  map[string]string "Validation Failed: Invalid phone format or missing prefix"
// @Failure      429      {object}  map[string]string "Rate limit exceeded: Try again in 1 minute"
// @Failure      503      {object}  map[string]string "Service Unavailable: SMS provider or Redis failure"
// @Failure      500      {object}  map[string]string "Internal Server Error"
// @Router       /api/v1/accounts/auth/send-otp [post]
func (h *handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOTPRequest

	fmt.Println("SendOTP handler invoked")
	// 1. Decode JSON
	if err := json.Decode(r, &req); err != nil {
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
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternal)
		return
	}

	// Hash the OTP before saving to Redis
	hash := sha256.Sum256([]byte(otp))
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
// @Description  Verifies the OTP hash from Redis. If successful, creates/updates the account in the database and returns a signed JWT.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        request  body      verifyOTPRequest  true  "Phone (+E.164) and 6-digit numeric OTP"
// @Success      200      {object}  verifyOTPResponse
// @Failure      400      {object}  map[string]string "Invalid JSON"
// @Failure      401      {object}  map[string]string "Unauthorized: Invalid or expired OTP"
// @Failure      422      {object}  map[string]string "Validation Failed: Non-numeric OTP or malformed phone"
// @Failure      500      {object}  map[string]string "Internal Server Error"
// @Router       /api/v1/accounts/auth/verify-otp [post]
func (h *handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyOTPRequest
	if err := json.Decode(r, &req); err != nil {
		json.WriteError(w, http.StatusBadRequest, constants.ErrInvalidJSON)
		return
	}

	// 1. Validate the DTO
	if err := h.validate.Struct(req); err != nil {
		json.WriteError(w, http.StatusUnprocessableEntity, constants.ErrInvalidPhoneOrCode)
		return
	}

	ctx := r.Context()
	otpKey := fmt.Sprintf("otp:%s", req.Phone)

	// 2. Fetch Hash from Redis
	storedHash, err := h.cache.Get(ctx, otpKey).Result()
	if err == redis.Nil {
		json.WriteError(w, http.StatusUnauthorized, constants.ErrInvalidOTP)
		return
	} else if err != nil {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternal)
		return
	}

	// 3. Verify Hash
	inputHash := sha256.Sum256([]byte(req.OTP))
	if fmt.Sprintf("%x", inputHash) != storedHash {
		json.WriteError(w, http.StatusUnauthorized, constants.ErrInvalidOTP)
		return
	}

	// 4. Success - Clear OTP and update DB
	h.cache.Del(ctx, otpKey)
	dbAccount, err := h.service.UpsertByPhone(ctx, req.Phone)
	if err != nil {
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternal)
		return
	}

	// 5. Generate a real JWT
	token, err := h.auth.GenerateToken(dbAccount.ID.String())
	if err != nil {
		h.logger.Error("failed to generate token", "error", err)
		json.WriteError(w, http.StatusInternalServerError, constants.ErrInternal)
		return
	}

	// 6. Build final response
	res := verifyOTPResponse{
		Message: constants.MsgOTPVerified,
		Token:   token,
		Account: MapAccountRow(dbAccount),
	}

	json.Write(w, http.StatusOK, res)
}
