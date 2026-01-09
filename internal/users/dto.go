// internal/users/dto.go
package users

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yabeye/addis_verify_backend/internal/account"
	repo "github.com/yabeye/addis_verify_backend/internal/database"
	"github.com/yabeye/addis_verify_backend/pkg/json"
)

type updateProfileRequest struct {
	FirstName   string     `json:"first_name" validate:"required"`
	MiddleName  string     `json:"middle_name"`
	LastName    string     `json:"last_name" validate:"required"`
	AliasName   string     `json:"alias_name"`
	Birthdate   *time.Time `json:"birthdate"`
	Gender      string     `json:"gender"`
	Citizenship string     `json:"citizenship"`
	Email       string     `json:"email" validate:"omitempty,email"`
	HeadshotURL string     `json:"headshot_url"`
	GovIDURL    string     `json:"gov_id_url"`
	PassportURL string     `json:"passport_url"`
	Address     addressDTO `json:"address"`
}

func (req *updateProfileRequest) Bind(r *http.Request) error {
	return json.Read(r, req)
}

type addressDTO struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	City    string `json:"city"`
	Zone    string `json:"zone"`
	Wereda  string `json:"wereda"`
	Kebele  string `json:"kebele"`
}

type FullUserProfileDTO struct {
	Account account.AccountDTO `json:"account"`
	Profile *UserProfileDTO    `json:"profile,omitempty"`
}

type UserProfileDTO struct {
	UserID            string     `json:"user_id"`
	FirstName         string     `json:"first_name"`
	MiddleName        string     `json:"middle_name"`
	LastName          string     `json:"last_name"`
	Birthdate         *time.Time `json:"birthdate"`
	Email             string     `json:"email"`
	UserHeadShotImage string     `json:"user_head_shot_image"`
	GovernmentIDImage string     `json:"government_id_image"`
	PassportImage     string     `json:"passport_image"`
	Address           addressDTO `json:"address"`
}

func UUIDToString(pgUUID pgtype.UUID) string {
	if !pgUUID.Valid {
		return ""
	}
	return uuid.UUID(pgUUID.Bytes).String()
}

func mapUserRowToDTO(u repo.GetUserWithAddressByAccountIDRow) *UserProfileDTO {
	if !u.UserID.Valid {
		return nil
	}
	profile := &UserProfileDTO{
		UserID:    UUIDToString(u.UserID),
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
	if u.MiddleName.Valid {
		profile.MiddleName = u.MiddleName.String
	}
	if u.Email.Valid {
		profile.Email = u.Email.String
	}
	if u.UserHeadShotImage.Valid {
		profile.UserHeadShotImage = u.UserHeadShotImage.String
	}
	if u.GovernmentIDImage.Valid {
		profile.GovernmentIDImage = u.GovernmentIDImage.String
	}
	if u.PassportImage.Valid {
		profile.PassportImage = u.PassportImage.String
	}
	if u.Birthdate.Valid {
		t := u.Birthdate.Time
		profile.Birthdate = &t
	}
	profile.Address = addressDTO{
		Country: u.Country.String, Region: u.Region.String, City: u.City.String,
		Zone: u.Zone.String, Wereda: u.Wereda.String, Kebele: u.Kebele.String,
	}
	return profile
}
