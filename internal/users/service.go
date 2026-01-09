package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yabeye/addis_verify_backend/internal/account"
	repo "github.com/yabeye/addis_verify_backend/internal/database"
)

type Service interface {
	GetFullMe(ctx context.Context, accountID pgtype.UUID) (*FullUserProfileDTO, error)
	UpdateFullProfile(ctx context.Context, accountID pgtype.UUID, req updateProfileRequest) error
}

type svc struct {
	repo repo.Querier
}

func New(repo repo.Querier) Service {
	return &svc{
		repo: repo,
	}
}

func (s *svc) GetFullMe(ctx context.Context, accountID pgtype.UUID) (*FullUserProfileDTO, error) {
	acc, err := s.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	response := &FullUserProfileDTO{
		Account: account.MapAccountRow(acc),
	}

	profileRow, err := s.repo.GetUserWithAddressByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return response, nil
		}
		return nil, err
	}

	response.Profile = mapUserRowToDTO(profileRow)
	return response, nil
}

func (s *svc) UpdateFullProfile(ctx context.Context, accountID pgtype.UUID, req updateProfileRequest) error {
	// 1. SECURITY CHECK: Verify Ownership
	// We ensure the URL contains the user's UUID so they can't link someone else's files.
	idStr := UUIDToString(accountID)

	if req.HeadshotURL != "" && !strings.Contains(req.HeadshotURL, idStr) {
		return fmt.Errorf("security violation: headshot file does not belong to this account")
	}
	if req.GovIDURL != "" && !strings.Contains(req.GovIDURL, idStr) {
		return fmt.Errorf("security violation: government ID file does not belong to this account")
	}
	if req.PassportURL != "" && !strings.Contains(req.PassportURL, idStr) {
		return fmt.Errorf("security violation: passport file does not belong to this account")
	}

	// 2. DATA PREPARATION
	dbBirthdate := pgtype.Date{Valid: false}
	if req.Birthdate != nil {
		dbBirthdate = pgtype.Date{Time: *req.Birthdate, Valid: true}
	}

	// User Profile Params
	userParams := repo.UpsertUserParams{
		AccountID:         accountID,
		FirstName:         req.FirstName,
		MiddleName:        pgtype.Text{String: req.MiddleName, Valid: req.MiddleName != ""},
		LastName:          req.LastName,
		AliasName:         pgtype.Text{String: req.AliasName, Valid: req.AliasName != ""},
		Birthdate:         dbBirthdate,
		Gender:            pgtype.Text{String: req.Gender, Valid: req.Gender != ""},
		Citizenship:       pgtype.Text{String: req.Citizenship, Valid: req.Citizenship != ""},
		Email:             pgtype.Text{String: req.Email, Valid: req.Email != ""},
		UserHeadShotImage: pgtype.Text{String: req.HeadshotURL, Valid: req.HeadshotURL != ""},
		GovernmentIDImage: pgtype.Text{String: req.GovIDURL, Valid: req.GovIDURL != ""},
		PassportImage:     pgtype.Text{String: req.PassportURL, Valid: req.PassportURL != ""},
	}

	// 3. EXECUTE UPDATES
	if _, err := s.repo.UpsertUser(ctx, userParams); err != nil {
		return fmt.Errorf("upsert user failed: %w", err)
	}

	// Address Params
	addrParams := repo.UpsertAddressParams{
		AccountID: accountID,
		Country:   req.Address.Country,
		Region:    pgtype.Text{String: req.Address.Region, Valid: req.Address.Region != ""},
		City:      pgtype.Text{String: req.Address.City, Valid: req.Address.City != ""},
		Zone:      pgtype.Text{String: req.Address.Zone, Valid: req.Address.Zone != ""},
		Wereda:    pgtype.Text{String: req.Address.Wereda, Valid: req.Address.Wereda != ""},
		Kebele:    pgtype.Text{String: req.Address.Kebele, Valid: req.Address.Kebele != ""},
	}

	if _, err := s.repo.UpsertAddress(ctx, addrParams); err != nil {
		return fmt.Errorf("upsert address failed: %w", err)
	}

	return nil
}
