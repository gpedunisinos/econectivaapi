package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	ErrGoogleConfig          = errors.New("invalid google config")
	ErrGoogleTokenInvalid    = errors.New("invalid google token")
	ErrGoogleAudienceInvalid = errors.New("invalid google audience")
	ErrGoogleUnavailable     = errors.New("google tokeninfo unavailable")
)

type GoogleTokenInfo struct {
	Audience string `json:"aud"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type googleTokenInfoResponse struct {
	Audience string `json:"aud"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

func ValidateGoogleIDToken(idToken string) (*GoogleTokenInfo, error) {
	return validateGoogleToken("id_token", idToken)
}

func ValidateGoogleAccessToken(accessToken string) (*GoogleTokenInfo, error) {
	return validateGoogleToken("access_token", accessToken)
}

func validateGoogleToken(tokenField string, tokenValue string) (*GoogleTokenInfo, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_WEB_CLIENT_ID"))
	if clientID == "" {
		return nil, ErrGoogleConfig
	}

	tokenValue = strings.TrimSpace(tokenValue)
	if tokenValue == "" {
		return nil, ErrGoogleTokenInvalid
	}

	tokenInfoURL := strings.TrimSpace(os.Getenv("GOOGLE_TOKENINFO_URL"))
	if tokenInfoURL == "" {
		tokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"
	}

	requestURL := tokenInfoURL + "?" + tokenField + "=" + url.QueryEscape(tokenValue)
	client := http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(requestURL)
	if err != nil {
		return nil, ErrGoogleUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, ErrGoogleTokenInvalid
	}

	var tokenInfo googleTokenInfoResponse
	if err = json.NewDecoder(response.Body).Decode(&tokenInfo); err != nil {
		return nil, ErrGoogleTokenInvalid
	}

	if strings.TrimSpace(tokenInfo.Audience) != clientID {
		return nil, ErrGoogleAudienceInvalid
	}

	if strings.TrimSpace(tokenInfo.Email) == "" {
		return nil, ErrGoogleTokenInvalid
	}

	return &GoogleTokenInfo{
		Audience: strings.TrimSpace(tokenInfo.Audience),
		Email:    strings.ToLower(strings.TrimSpace(tokenInfo.Email)),
		Name:     strings.TrimSpace(tokenInfo.Name),
	}, nil
}
