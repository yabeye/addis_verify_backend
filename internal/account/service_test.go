package account

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	repo "github.com/yabeye/addis_verify_backend/internal/database"
)

// 1. Create a Mock for the Querier
type mockQuerier struct {
	mock.Mock
}

func (m *mockQuerier) GetAccountByID(ctx context.Context, id pgtype.UUID) (repo.Account, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(repo.Account), args.Error(1)
}

func (m *mockQuerier) GetAccountByPhone(ctx context.Context, phone string) (repo.Account, error) {
	args := m.Called(ctx, phone)
	return args.Get(0).(repo.Account), args.Error(1)
}

func (m *mockQuerier) UpsertAccount(ctx context.Context, phone string) (repo.Account, error) {
	args := m.Called(ctx, phone)
	return args.Get(0).(repo.Account), args.Error(1)
}

func (m *mockQuerier) UpdateAccountStatus(ctx context.Context, params repo.UpdateAccountStatusParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

// 2. Test Suites

func TestGetAccountByID(t *testing.T) {
	ctx := context.Background()
	mockID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}

	t.Run("account found", func(t *testing.T) {
		mockRepo := new(mockQuerier)
		expected := repo.Account{ID: mockID, Phone: "+251911223344"}

		mockRepo.On("GetAccountByID", ctx, mockID).Return(expected, nil)

		service := New(mockRepo)
		acc, err := service.GetAccountByID(ctx, mockID)

		assert.NoError(t, err)
		assert.Equal(t, expected.Phone, acc.Phone)
		mockRepo.AssertExpectations(t)
	})

	t.Run("account not found", func(t *testing.T) {
		mockRepo := new(mockQuerier)
		mockRepo.On("GetAccountByID", ctx, mockID).Return(repo.Account{}, errors.New("not found"))

		service := New(mockRepo)
		_, err := service.GetAccountByID(ctx, mockID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestUpsertByPhone(t *testing.T) {
	ctx := context.Background()
	testPhone := "+251911223344"

	t.Run("success", func(t *testing.T) {
		mockRepo := new(mockQuerier)
		expectedAccount := repo.Account{
			Phone:  testPhone,
			Status: repo.AccountStatusActive,
		}

		mockRepo.On("UpsertAccount", ctx, testPhone).Return(expectedAccount, nil)

		service := New(mockRepo)
		acc, err := service.UpsertByPhone(ctx, testPhone)

		assert.NoError(t, err)
		assert.Equal(t, testPhone, acc.Phone)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateAccountStatus(t *testing.T) {
	ctx := context.Background()
	mockID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}

	t.Run("successfully update status", func(t *testing.T) {
		mockRepo := new(mockQuerier)
		params := repo.UpdateAccountStatusParams{
			ID:     mockID,
			Status: repo.AccountStatusActive,
		}

		mockRepo.On("UpdateAccountStatus", ctx, params).Return(nil)

		service := New(mockRepo)
		err := service.UpdateAccountStatus(ctx, mockID, repo.AccountStatusActive)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
