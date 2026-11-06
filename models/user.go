package models

import (
	"time"

	"github.com/OdairPianta/unisinos_usp_api/enums"
	"gorm.io/gorm"
)

type User struct {
	UsageMode *string `gorm:"size:16" json:"usage_mode"`
	DefaultModel
	DeletedAt          gorm.DeletedAt        `gorm:"index" json:"deleted_at"`
	Name               string                `gorm:"size:255;notNull" validate:"required" json:"name"`
	Email              string                `gorm:"size:255;notNull;unique" json:"email"`
	EmailVerifiedAt    *time.Time            `json:"email_verified_at"`
	Password           string                `gorm:"size:255" validate:"required" json:"password"`
	RememberToken      string                `gorm:"size:255" json:"remember_token"`
	FCMToken           string                `gorm:"size:255" json:"fcm_token"`
	Token              string                `gorm:"size:255" json:"token"`
	ResetPasswordToken string                `gorm:"size:255" json:"reset_password_token"`
	Profile            enums.UserProfileEnum `gorm:"notNull" validate:"required" json:"profile"` // 10 - Usuário, 40 - Administrador
	Phone              string                `gorm:"size:255" json:"phone"`
	IsGoogleAuth       bool                  `gorm:"default:false" json:"is_google_auth"`
	IsAppleAuth        bool                  `gorm:"default:false" json:"is_apple_auth"`
}
