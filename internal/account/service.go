package account

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	repo "github.com/yabeye/addis_verify_backend/internal/database"
)

// Service defines the exported behavior of the account module
type Service interface {
	GetAccountByPhone(ctx context.Context, phone string) (repo.Account, error)
	UpdateAccountStatus(ctx context.Context, id pgtype.UUID, status repo.AccountStatus) error
	UpsertByPhone(ctx context.Context, phone string) (repo.Account, error)
}

type svc struct {
	repo repo.Querier
}

// New creates a new account service implementation
func New(repo repo.Querier) Service {
	return &svc{
		repo: repo,
	}
}

func (s *svc) GetAccountByPhone(ctx context.Context, phone string) (repo.Account, error) {
	return s.repo.GetAccountByPhone(ctx, phone)
}

func (s *svc) UpdateAccountStatus(ctx context.Context, id pgtype.UUID, status repo.AccountStatus) error {
	return s.repo.UpdateAccountStatus(ctx, repo.UpdateAccountStatusParams{
		ID:     id,
		Status: status,
	})
}

func (s *svc) UpsertByPhone(ctx context.Context, phone string) (repo.Account, error) {
	return s.repo.UpsertAccount(ctx, phone)
}
