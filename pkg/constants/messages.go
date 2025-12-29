package constants

// Success Messages
const (
	MsgOTPSent     = "OTP sent successfully"
	MsgOTPVerified = "OTP verified successfully"
)

// Error Messages
const (
	ErrInvalidJSON        = "Invalid request body"
	ErrInvalidPhone       = "A valid phone number in E.164 format (e.g. +251...) is required"
	ErrInvalidPhoneOrCode = "Invalid phone number or OTP code"
	ErrRateLimit          = "Please wait 60 seconds before requesting a new code"
	ErrInvalidOTP         = "Invalid OTP code"
	ErrInternal           = "Internal server error"
	ErrServiceUnavailable = "Service unavailable"
	ErrAccountSuspended   = "Your account has been suspended"
	ErrFailedToSendSMS    = "Failed to send SMS"
)
