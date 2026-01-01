package account

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	repo "github.com/yabeye/addis_verify_backend/internal/database"
	"github.com/yabeye/addis_verify_backend/pkg/auth"
	"github.com/yabeye/addis_verify_backend/pkg/messenger"
)

// --- Mocks ---

type mockService struct{ mock.Mock }

func (m *mockService) GetAccountByID(ctx context.Context, id pgtype.UUID) (repo.Account, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(repo.Account), args.Error(1)
}
func (m *mockService) GetAccountByPhone(ctx context.Context, p string) (repo.Account, error) {
	args := m.Called(ctx, p)
	return args.Get(0).(repo.Account), args.Error(1)
}
func (m *mockService) UpdateAccountStatus(ctx context.Context, id pgtype.UUID, s repo.AccountStatus) error {
	return m.Called(ctx, id, s).Error(0)
}
func (m *mockService) UpsertByPhone(ctx context.Context, p string) (repo.Account, error) {
	args := m.Called(ctx, p)
	return args.Get(0).(repo.Account), args.Error(1)
}

type mockAuth struct{ mock.Mock }

// Updated to use auth.TokenDetails to match your manager.go
func (m *mockAuth) GenerateTokenPair(id string, iat time.Time) (*auth.TokenDetails, error) {
	args := m.Called(id, iat)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenDetails), args.Error(1)
}

func (m *mockAuth) VerifyToken(token string) (*auth.Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Claims), args.Error(1)
}

type mockMessenger struct{ mock.Mock }

func (m *mockMessenger) Send(ctx context.Context, msg messenger.Message) error {
	return m.Called(ctx, msg).Error(0)
}

// --- Test Suite ---

func TestHandler_VerifyOTP(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := new(mockService)
	authMgr := new(mockAuth)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	pepper := "test-pepper"

	h := &handler{
		service:    svc,
		logger:     logger,
		cache:      rdb,
		validate:   validator.New(),
		auth:       authMgr,
		hashPepper: pepper,
	}

	t.Run("Successful Verification", func(t *testing.T) {
		phone := "+251911223344"
		otp := "123456"
		mockID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		now := time.Now()

		// 1. Seed Redis with peppered hash
		combined := phone + otp + pepper
		hash := sha256.Sum256([]byte(combined))
		mr.Set("otp:"+phone, fmt.Sprintf("%x", hash))

		// 2. Expectations
		svc.On("UpsertByPhone", mock.Anything, phone).Return(repo.Account{
			ID:             mockID,
			Phone:          phone,
			TokenValidFrom: pgtype.Timestamptz{Time: now, Valid: true},
		}, nil)

		authMgr.On("GenerateTokenPair", mockID.String(), mock.Anything).Return(&auth.TokenDetails{
			AccessToken:  "fake-access",
			RefreshToken: "fake-refresh",
		}, nil)

		body, _ := json.Marshal(map[string]string{"phone": phone, "otp": otp})
		req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.VerifyOTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, mr.Exists("otp:"+phone)) // Redis clean
	})
}

func TestHandler_RefreshToken_Security(t *testing.T) {
	svc := new(mockService)
	authMgr := new(mockAuth)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	h := &handler{
		service:  svc,
		auth:     authMgr,
		logger:   logger,
		validate: validator.New(),
	}

	t.Run("Failure: Reject Access Token on Refresh Endpoint", func(t *testing.T) {
		accessToken := "valid-access-token-but-wrong-type"
		mockID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}

		// 1. Mock a valid JWT but with Type "access"
		authMgr.On("VerifyToken", accessToken).Return(&auth.Claims{
			AccountID: mockID.String(),
			Type:      "access", // <--- This triggers the 403 Forbidden
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt: jwt.NewNumericDate(time.Now()),
			},
		}, nil)

		body, _ := json.Marshal(map[string]string{"refresh_token": accessToken})

		// Corrected path to match your MountRoutes logic
		req := httptest.NewRequest(http.MethodPost, "/accounts/auth/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// 2. Execute the handler
		h.RefreshToken(w, req)

		// 3. Assertions
		assert.Equal(t, http.StatusForbidden, w.Code)

		// Verify that we didn't touch the database or generate new tokens
		svc.AssertNotCalled(t, "GetAccountByID", mock.Anything, mock.Anything)
		authMgr.AssertNotCalled(t, "GenerateTokenPair", mock.Anything, mock.Anything)
	})
}

func TestHandler_RefreshToken(t *testing.T) {
	svc := new(mockService)
	authMgr := new(mockAuth)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	h := &handler{service: svc, auth: authMgr, logger: logger, validate: validator.New()}

	t.Run("Successful Rotation", func(t *testing.T) {
		token := "old-refresh-token"
		mockID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		now := time.Now()

		// FIX: Use jwt.NumericDate and initialize the embedded RegisteredClaims properly
		authMgr.On("VerifyToken", token).Return(&auth.Claims{
			AccountID: mockID.String(),
			Type:      "refresh",
			// Initialize the embedded struct fields directly
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt: jwt.NewNumericDate(now.Add(-time.Minute)),
			},
		}, nil)

		// ... rest of the test code ...
		svc.On("GetAccountByID", mock.Anything, mockID).Return(repo.Account{
			Phone:          "+251911223344",
			TokenValidFrom: pgtype.Timestamptz{Time: now.Add(-5 * time.Minute), Valid: true},
		}, nil)

		svc.On("UpsertByPhone", mock.Anything, "+251911223344").Return(repo.Account{
			ID:             mockID,
			TokenValidFrom: pgtype.Timestamptz{Time: now, Valid: true},
		}, nil)

		authMgr.On("GenerateTokenPair", mockID.String(), mock.Anything).Return(&auth.TokenDetails{
			AccessToken: "new-access", RefreshToken: "new-refresh",
		}, nil)

		body, _ := json.Marshal(map[string]string{"refresh_token": token})
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.RefreshToken(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_Logout(t *testing.T) {
	svc := new(mockService)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	h := &handler{service: svc, logger: logger}

	t.Run("Logout Success", func(t *testing.T) {
		mockIDStr := "550e8400-e29b-41d4-a716-446655440000"
		var mockID pgtype.UUID
		mockID.Scan(mockIDStr)

		// Inject Context
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		ctx := context.WithValue(req.Context(), "user_id", mockIDStr)
		req = req.WithContext(ctx)

		svc.On("GetAccountByID", mock.Anything, mockID).Return(repo.Account{Phone: "+251911223344"}, nil)
		svc.On("UpsertByPhone", mock.Anything, "+251911223344").Return(repo.Account{}, nil)

		w := httptest.NewRecorder()
		h.Logout(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
