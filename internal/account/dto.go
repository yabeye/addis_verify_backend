package account

import (
	"time"

	repo "github.com/yabeye/addis_verify_backend/internal/database"
)

// sendOTPRequest represents the payload for requesting a new OTP
// @Name SendOTPRequest
type sendOTPRequest struct {
	// Phone must be in E.164 format and include the '+' prefix
	Phone string `json:"phone" validate:"required,e164,startswith=+" example:"+251911223344"`
}

// sendOTPResponse represents the success message after an OTP is triggered
// @Name SendOTPResponse
type sendOTPResponse struct {
	Message string `json:"message" example:"OTP sent successfully"`
}

// verifyOTPRequest represents the payload to exchange an OTP for a JWT
// @Name VerifyOTPRequest
type verifyOTPRequest struct {
	Phone string `json:"phone" validate:"required,e164,startswith=+" example:"+251911223344"`
	// OTP must be exactly 6 digits
	OTP string `json:"otp" validate:"required,len=6,numeric" example:"123456"`
}

// authSuccessResponse contains the authentication token and user profile
// @Name AuthSuccessResponse
type authSuccessResponse struct {
	Message      string     `json:"message" example:"OTP verified successfully"`
	AccessToken  string     `json:"access_token" example:"eyJhbGciOiJIUzI1Ni..."`
	RefreshToken string     `json:"refresh_token" example:"eyJhbGciOiJIUzI1Ni..."`
	Account      AccountDTO `json:"account"`
}

// AccountDTO represents the public-facing account profile
// @Name AccountDTO
type AccountDTO struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Phone     string `json:"phone" example:"+251911223344"`
	Status    string `json:"status" example:"active"`
	UpdatedAt string `json:"updated_at" example:"2023-10-27T10:00:00Z"`
	CreatedAt string `json:"created_at" example:"2023-10-27T10:00:00Z"`
}

// MapAccountRow translates the database record into a clean API response
func MapAccountRow(u repo.Account) AccountDTO {
	return AccountDTO{
		ID:        u.ID.String(),
		Phone:     u.Phone,
		Status:    string(u.Status),
		UpdatedAt: u.UpdatedAt.Time.Format(time.RFC3339),
		CreatedAt: u.CreatedAt.Time.Format(time.RFC3339),
	}
}
