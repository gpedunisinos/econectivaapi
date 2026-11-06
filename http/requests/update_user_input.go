package requests

import (
	"github.com/OdairPianta/unisinos_usp_api/enums"
)

type UpdateUserInput struct {
	Name               string                `json:"name" binding:"required"`
	Email              string                `json:"email" binding:"required,email"`
	Password           string                `json:"password"`
	RememberToken      string                `json:"remember_token"`
	FCMToken           string                `json:"fcm_token"`
	ResetPasswordToken string                `json:"reset_password_token"`
	Profile            enums.UserProfileEnum `json:"profile" binding:"required,oneof=10 40"`
	Phone              *string               `json:"phone"`
}
