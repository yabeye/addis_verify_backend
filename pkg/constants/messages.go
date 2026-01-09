package constants

// Success Messages
const (
	MsgOTPSent     = "OTP sent successfully"
	MsgOTPVerified = "OTP verified successfully"
)

// Error Messages
const (
	// auth errors
	ErrInvalidJSON           = "Invalid request body"
	ErrInvalidPhone          = "A valid phone number in E.164 format (e.g. +251...) is required"
	ErrInvalidPhoneOrCode    = "Invalid phone number or OTP code"
	ErrRateLimit             = "Please wait 60 seconds before requesting a new code"
	ErrInvalidOTP            = "Invalid or Expired OTP code"
	ErrFailedToSendSMS       = "Failed to send SMS"
	ErrInvalidOrExpiredToken = "Invalid or expired refresh token"

	ErrAccountSuspended    = "Your account has been suspended"
	ErrAccountNotFound     = "Account not found"
	ErrUnauthorizedError       = "Not authorized"
	ErrServiceUnavailable  = "Service unavailable"
	ErrInternalServerError = "Internal server error"
)
