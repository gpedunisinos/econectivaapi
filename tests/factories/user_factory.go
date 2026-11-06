package factories

import (
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateUser() (*models.User, error) {
	verifiedAt := time.Now()
	model := &models.User{
		Name:               fakers.RandomString(),
		Email:              fakers.Email(),
		Password:           fakers.Word(),
		EmailVerifiedAt:    &verifiedAt,
		RememberToken:      fakers.UUID(),
		FCMToken:           fakers.UUID(),
		Token:              fakers.UUID(),
		ResetPasswordToken: fakers.UUID(),
		Profile:            enums.UserProfileEnumAdministrator,
		Phone:              "+5511988776655",
	}

	err := config.DB.Create(&model).Error
	if err != nil {
		return nil, err
	}

	return model, nil
}
