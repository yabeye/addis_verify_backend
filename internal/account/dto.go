package account

import (
	"time"

	repo "github.com/yabeye/addis_verify_backend/internal/database"
)

type sendOTPRequest struct {
	Phone string `json:"phone" validate:"required,e164,startswith=+" example:"+251911223344"`
}

type sendOTPResponse struct {
	Message string `json:"message" example:"OTP sent successfully"`
}

type verifyOTPRequest struct {
	Phone string `json:"phone" validate:"required,e164,startswith=+" example:"+251911223344"`
	OTP   string `json:"otp" validate:"required,len=6,numeric" example:"123456"`
}

type verifyOTPResponse struct {
	Message string     `json:"message" example:"OTP verified successfully"`
	Token   string     `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Account accountDTO `json:"account"`
}

type accountDTO struct {
	ID        string `json:"id"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
	CreatedAt string `json:"created_at"`
}

// MapAccountRow translates the database record into a clean API response
func MapAccountRow(u repo.Account) accountDTO {
	return accountDTO{
		ID:        u.ID.String(),
		Phone:     u.Phone,
		Status:    string(u.Status),
		UpdatedAt: u.UpdatedAt.Time.Format(time.RFC3339),
		CreatedAt: u.CreatedAt.Time.Format(time.RFC3339),
	}
}
