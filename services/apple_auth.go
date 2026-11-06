package services

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services/token"
	"github.com/dgrijalva/jwt-go"
	"gorm.io/gorm"
)

func AppleAuth(idToken string, name string) (*models.User, string, error) {
	parsedToken, err := jwt.Parse(idToken, applePublicKeyFunc)
	if err != nil || !parsedToken.Valid {
		return nil, "", errors.New("invalid Apple token")
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, "", errors.New("invalid Apple token claims")
	}

	if claims["iss"] != "https://appleid.apple.com" {
		return nil, "", errors.New("invalid Apple issuer")
	}

	if !appleAudienceMatches(claims["aud"]) {
		return nil, "", errors.New("invalid Apple audience")
	}

	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return nil, "", errors.New("email not found in Apple token")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = extractAppleUserName(claims, email)
	}

	user := &models.User{}
	err = config.DB.Where("email = ?", email).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = &models.User{
				Name:        name,
				Email:       email,
				IsAppleAuth: true,
				Profile:     enums.UserProfileEnumUser,
			}
			if err := config.DB.Create(user).Error; err != nil {
				return nil, "", err
			}
		} else {
			return nil, "", err
		}
	}

	jwtToken, err := token.GenerateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, jwtToken, nil
}

func applePublicKeyFunc(token *jwt.Token) (interface{}, error) {
	kid, _ := token.Header["kid"].(string)
	if kid == "" || token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
		return nil, errors.New("invalid Apple token header")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://appleid.apple.com/auth/keys")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple public keys returned status %d", resp.StatusCode)
	}

	var keys struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, err
	}

	for _, key := range keys.Keys {
		if key.Kid != kid || key.Alg != jwt.SigningMethodRS256.Alg() || key.Kty != "RSA" {
			continue
		}

		return buildAppleRSAPublicKey(key.N, key.E)
	}

	return nil, errors.New("Apple public key not found")
}

func buildAppleRSAPublicKey(modulus string, exponent string) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}

	exponentBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil {
		return nil, err
	}

	publicKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulusBytes),
		E: int(new(big.Int).SetBytes(exponentBytes).Int64()),
	}
	if publicKey.N.Sign() <= 0 || publicKey.E <= 1 {
		return nil, errors.New("invalid Apple public key")
	}

	return publicKey, nil
}

func appleAudienceMatches(audienceClaim interface{}) bool {
	expectedAudiences := []string{
		os.Getenv("APPLE_CLIENT_ID"),
		os.Getenv("APPLE_CLIENT_ID_IOS"),
		os.Getenv("APPLE_CLIENT_ID_WEB"),
		os.Getenv("APPLE_SERVICE_ID"),
	}

	var allowedAudiences []string
	for _, audience := range expectedAudiences {
		audience = strings.TrimSpace(audience)
		if audience != "" {
			allowedAudiences = append(allowedAudiences, audience)
		}
	}

	if len(allowedAudiences) == 0 {
		return true
	}

	switch audience := audienceClaim.(type) {
	case string:
		return stringInSlice(audience, allowedAudiences)
	case []interface{}:
		for _, item := range audience {
			audienceValue, ok := item.(string)
			if ok && stringInSlice(audienceValue, allowedAudiences) {
				return true
			}
		}
	}

	return false
}

func extractAppleUserName(claims jwt.MapClaims, email string) string {
	if name, ok := claims["name"].(string); ok && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}

	if givenName, ok := claims["given_name"].(string); ok && strings.TrimSpace(givenName) != "" {
		if familyName, ok := claims["family_name"].(string); ok && strings.TrimSpace(familyName) != "" {
			return strings.TrimSpace(fmt.Sprintf("%s %s", givenName, familyName))
		}
		return strings.TrimSpace(givenName)
	}

	emailParts := strings.Split(email, "@")
	if len(emailParts) > 0 && strings.TrimSpace(emailParts[0]) != "" {
		return strings.TrimSpace(emailParts[0])
	}

	return "Apple User"
}

func stringInSlice(value string, values []string) bool {
	for _, currentValue := range values {
		if currentValue == value {
			return true
		}
	}

	return false
}
