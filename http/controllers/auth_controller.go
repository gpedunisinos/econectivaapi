package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/notifications"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/OdairPianta/unisinos_usp_api/services/token"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// @Summary		Register an user
// @Description	Register public user by json
// @Tags			users
// @Accept		json
// @Produce		json
// @Param			user	body		requests.RegisterInput	true	"Register user"
// @Success		200		{object}	models.User
// @Failure 400 {object} models.APIMessage "Bad request"
// @Failure 500 {object} models.APIMessage "Internal server error"
// @Router			/register [post]
func Register(context *gin.Context) {
	var input requests.RegisterInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Password = strings.TrimSpace(input.Password)

	registeredUser, errGetUser := repository.FindUserByEmail(input.Email)
	if errGetUser == nil {
		if registeredUser.IsGoogleAuth {
			context.JSON(http.StatusBadRequest, gin.H{"message": "Este email já está vinculado a uma conta Google. Entre com Google para continuar."})
			return
		}

		context.JSON(http.StatusBadRequest, gin.H{"message": "Email já cadastrado!"})
		return
	}

	if !errors.Is(errGetUser, gorm.ErrRecordNotFound) {
		sentry.CaptureException(errGetUser)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao verificar usuário"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criptografar senha!"})
		return
	}

	user := models.User{
		Name:         input.Name,
		Email:        input.Email,
		Password:     string(hashedPassword),
		Profile:      enums.UserProfileEnumUser,
		IsGoogleAuth: false,
	}

	if err = repository.CreateUser(&user); err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criar usuário"})
		return
	}

	tokenValue, loginErr := token.LoginCheck(user.Email, input.Password)
	if loginErr != nil {
		sentry.CaptureException(loginErr)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Usuário criado, mas não foi possível gerar autenticação."})
		return
	}

	user.Token = tokenValue
	context.JSON(http.StatusOK, user)
}

// @Summary		Login an user
// @Description	Login by json user
// @Tags			users
// @Accept		json
// @Produce		json
// @Param			user	body		requests.LoginInput	true	"Add user"
// @Success		200		{object}	models.User
// @Failure 400 {object} models.APIMessage "Bad request"
// @Failure 404 {object} models.APIMessage "Not found"
// @Router			/login [post]
func Login(context *gin.Context) {

	var input requests.LoginInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Password = strings.TrimSpace(input.Password)

	registeredUser, errGetUser := repository.FindUserByEmail(input.Email)
	if errGetUser == nil && registeredUser.IsGoogleAuth {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Este email está vinculado a uma conta Google. Entre com Google para continuar."})
		return
	}

	user := models.User{}

	user.Email = input.Email
	user.Password = input.Password

	token, loginErr := token.LoginCheck(user.Email, user.Password)

	if loginErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Email ou senha incorretos!"})
		return
	}
	registeredUser, errGetUser = repository.FindUserByEmail(input.Email)
	if errGetUser != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Usuário não encontrado"})
		return
	}
	registeredUser.Token = token

	context.JSON(http.StatusOK, registeredUser)
}

// @Summary		Login with Google
// @Description	Login by Google id_token or access_token
// @Tags			users
// @Accept		json
// @Produce		json
// @Param			user	body		requests.LoginGoogleInput	true	"Google id/access token"
// @Success		200		{object}	models.User
// @Failure 400 {object} models.APIMessage "Bad request"
// @Failure 401 {object} models.APIMessage "Unauthorized"
// @Router			/login/google [post]
func LoginGoogle(context *gin.Context) {
	var input requests.LoginGoogleInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	if strings.TrimSpace(input.IDToken) == "" && strings.TrimSpace(input.AccessToken) == "" {
		context.JSON(http.StatusBadRequest, gin.H{"message": "O campo IDToken ou AccessToken é obrigatório."})
		return
	}

	var (
		tokenInfo *services.GoogleTokenInfo
		err       error
	)

	if strings.TrimSpace(input.IDToken) != "" {
		tokenInfo, err = services.ValidateGoogleIDToken(input.IDToken)
	} else {
		tokenInfo, err = services.ValidateGoogleAccessToken(input.AccessToken)
	}
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGoogleConfig):
			context.JSON(http.StatusInternalServerError, gin.H{"message": "Configuração do Google inválida"})
		case errors.Is(err, services.ErrGoogleUnavailable):
			context.JSON(http.StatusBadGateway, gin.H{"message": "Não foi possível validar o token do Google"})
		case errors.Is(err, services.ErrGoogleTokenInvalid), errors.Is(err, services.ErrGoogleAudienceInvalid):
			context.JSON(http.StatusUnauthorized, gin.H{"message": "Token do Google inválido"})
		default:
			context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao validar o token do Google"})
		}
		return
	}

	user, err := repository.FindUserByEmail(tokenInfo.Email)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			sentry.CaptureException(err)
			context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao buscar usuário"})
			return
		}

		passwordHash, errHash := bcrypt.GenerateFromPassword([]byte("google-"+time.Now().Format(time.RFC3339Nano)), bcrypt.DefaultCost)
		if errHash != nil {
			sentry.CaptureException(errHash)
			context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criptografar a senha!"})
			return
		}

		name := tokenInfo.Name
		if name == "" {
			name = strings.Split(tokenInfo.Email, "@")[0]
		}

		verifiedAt := time.Now()
		newUser := models.User{
			Name:            name,
			Email:           tokenInfo.Email,
			EmailVerifiedAt: &verifiedAt,
			Password:        string(passwordHash),
			Profile:         enums.UserProfileEnumUser,
			IsGoogleAuth:    true,
		}

		if errCreate := repository.CreateUser(&newUser); errCreate != nil {
			sentry.CaptureException(errCreate)
			context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criar usuário"})
			return
		}

		user = &newUser
	}

	jwtToken, tokenErr := token.GenerateToken(user.ID)
	if tokenErr != nil {
		sentry.CaptureException(tokenErr)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao gerar o token do usuário"})
		return
	}
	user.Token = jwtToken

	context.JSON(http.StatusOK, user)
}

// @Summary		Login an user using apple token
// @Description	Login an user using apple token
// @Tags		users
// @Accept		json
// @Produce		json
// @Param			user	body		requests.LoginAppleInput	true	"Login with apple token"
// @Success		200		{object}	models.User
// @Failure 400 {object} models.APIMessage "Bad request"
// @Failure 404 {object} models.APIMessage "Not found"
// @Router			/login/apple [post]
func LoginApple(context *gin.Context) {
	var input requests.LoginAppleInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	if input.IDToken == "" {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Você deve enviar o IDToken!"})
		return
	}

	user, token, err := services.AppleAuth(input.IDToken, input.Name)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Erro ao efetuar login!"})
		return
	}

	user.Token = token
	context.JSON(http.StatusOK, user)
}

// @Summary		Forgot password
// @Description	Forgot password by json user
// @Tags			users
// @Accept		json
// @Produce		json
// @Param			user	body		requests.ForgotPasswordInput	true	"Forgot password"
// @Success		200		{object}	string
// @Failure 400 {object} models.APIMessage "Bad request"
// @Failure 404 {object} models.APIMessage "Not found"
// @Router			/forgot_password [post]
func ForgotPassword(context *gin.Context) {
	var input requests.ForgotPasswordInput

	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	user, err := repository.FindUserByEmail(input.Email)

	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Usuário não encontrado"})
		return
	}

	token, err := token.GenerateToken(user.ID)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao gerar o token do usuário"})
		return
	}
	user.ResetPasswordToken = token

	err = repository.UpdateUser(user, &requests.UpdateUserInput{
		ResetPasswordToken: user.ResetPasswordToken,
	})
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao atualizar usuário"})
		return
	}

	notification := notifications.ForgotPasswordNotification{User: *user}
	err = notification.Send()

	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao enviar email!"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Email enviado com sucesso!"})
}

// @Summary		Recover password
// @Description	Recover password by json user
// @Tags			users
// @Accept		json
// @Produce		json
// @Param			user	body		requests.RecoverPasswordInput	true	"Recover password"
// @Success		200		{object}	models.User
// @Failure 400 {object} models.APIMessage "Bad request"
// @Failure 404 {object} models.APIMessage "Not found"
// @Router			/recover_password [post]
func RecoverPassword(context *gin.Context) {
	var input requests.RecoverPasswordInput

	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	user := models.User{}

	user.ResetPasswordToken = input.ResetPasswordToken
	user.Password = input.Password

	err := config.DB.Where("reset_password_token = ?", user.ResetPasswordToken).First(&user).Error
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Token não encontrado"})
		return
	}

	hashedPassword, errEncryptPass := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	if errEncryptPass != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao criptografar senha!"})
		return
	}

	user.Password = string(hashedPassword)
	user.ResetPasswordToken = ""

	err = config.DB.Model(&user).Updates(map[string]interface{}{"password": user.Password, "reset_password_token": user.ResetPasswordToken}).Error
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao atualizar usuário"})
		return
	}

	token, loginErr := token.LoginCheck(user.Email, input.Password)

	if loginErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "O email ou a senha estão incorretos."})
		return
	}

	user.Token = token

	context.JSON(http.StatusOK, user)
}
