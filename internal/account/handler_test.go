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

	"github.com/alicebob/miniredis/v2"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	repo "github.com/yabeye/addis_verify_backend/internal/database"
	"github.com/yabeye/addis_verify_backend/pkg/messenger"
)

// --- Mocks ---
type mockService struct{ mock.Mock }

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

type mockMessenger struct{ mock.Mock }

func (m *mockMessenger) Send(ctx context.Context, msg messenger.Message) error {
	return m.Called(ctx, msg).Error(0)
}

type mockAuth struct{ mock.Mock }

func (m *mockAuth) GenerateToken(id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}
func (m *mockAuth) ValidateToken(t string) (string, error) {
	args := m.Called(t)
	return args.String(0), args.Error(1)
}

// --- Test Suite ---

func TestHandler_SendOTP(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := new(mockService)
	msgr := new(mockMessenger)
	authMgr := new(mockAuth)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	h := &handler{
		service:   svc,
		logger:    logger,
		cache:     rdb,
		validate:  validator.New(),
		genOTP:    func() (string, error) { return "123456", nil },
		messenger: msgr,
		auth:      authMgr,
	}

	t.Run("Exhaustive Validation Tests", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"Totally Empty JSON", `{}`},
			{"Phone is Null", `{"phone": null}`},
			{"Phone is Empty String", `{"phone": ""}`},
			{"Phone is just spaces", `{"phone": "   "}`},
			{"Phone missing plus", `{"phone": "251911223344"}`},
			{"Phone invalid format", `{"phone": "+251-invalid"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/auth/send-otp", bytes.NewBufferString(tt.body))
				w := httptest.NewRecorder()
				h.SendOTP(w, req)
				assert.Equal(t, http.StatusUnprocessableEntity, w.Code, tt.name)
			})
		}
	})
}

func TestHandler_VerifyOTP(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := new(mockService)
	authMgr := new(mockAuth)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	h := &handler{
		service:  svc,
		logger:   logger,
		cache:    rdb,
		validate: validator.New(),
		auth:     authMgr,
	}

	t.Run("Matrix Edge Case Tests", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"All fields missing", `{}`},
			{"All fields null", `{"phone": null, "otp": null}`},
			{"All fields empty string", `{"phone": "", "otp": ""}`},

			{"Valid Phone / Missing OTP", `{"phone": "+251911223344"}`},
			{"Valid Phone / Null OTP", `{"phone": "+251911223344", "otp": null}`},
			{"Valid Phone / Empty OTP", `{"phone": "+251911223344", "otp": ""}`},

			{"Missing Phone / Valid OTP", `{"otp": "123456"}`},
			{"Null Phone / Valid OTP", `{"phone": null, "otp": "123456"}`},
			{"Empty Phone / Valid OTP", `{"phone": "", "otp": "123456"}`},

			{"Valid Phone / Non-numeric OTP", `{"phone": "+251911223344", "otp": "12A456"}`},
			{"Valid Phone / Short OTP", `{"phone": "+251911223344", "otp": "123"}`},
			{"Valid Phone / Long OTP", `{"phone": "+251911223344", "otp": "1234567"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/auth/verify-otp", bytes.NewBufferString(tt.body))
				w := httptest.NewRecorder()
				h.VerifyOTP(w, req)
				assert.Equal(t, http.StatusUnprocessableEntity, w.Code, tt.name)
			})
		}
	})

	t.Run("Business Logic: Successful Verification", func(t *testing.T) {
		phone := "+251911223344"
		otp := "123456"
		hash := sha256.Sum256([]byte(otp))
		mr.Set("otp:"+phone, fmt.Sprintf("%x", hash))

		mockID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		svc.On("UpsertByPhone", mock.Anything, phone).Return(repo.Account{ID: mockID, Phone: phone}, nil)
		authMgr.On("GenerateToken", mock.Anything).Return("fake-token", nil)

		body, _ := json.Marshal(verifyOTPRequest{Phone: phone, OTP: otp})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/auth/verify-otp", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		h.VerifyOTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, mr.Exists("otp:"+phone))
	})
}
