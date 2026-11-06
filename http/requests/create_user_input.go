package requests

import (
	"github.com/OdairPianta/unisinos_usp_api/enums"
)

type CreateUserInput struct {
	Name          string                `json:"name" binding:"required"`
	Email         string                `json:"email" binding:"required,email,not_exists=users.email"`
	Password      string                `json:"password" binding:"required"`
	RememberToken string                `json:"remember_token"`
	FCMToken      string                `json:"fcm_token"`
	Profile       enums.UserProfileEnum `json:"profile" binding:"required,oneof=10 40"`
	Phone         string                `json:"phone"`
}
