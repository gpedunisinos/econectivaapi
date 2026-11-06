package services

import (
	"encoding/base64"
	"testing"

	"github.com/dgrijalva/jwt-go"
)

func TestAppleAudienceAllowsTokenWithoutConfiguredClients(t *testing.T) {
	for _, key := range []string{"APPLE_CLIENT_ID", "APPLE_CLIENT_ID_IOS", "APPLE_CLIENT_ID_WEB", "APPLE_SERVICE_ID"} {
		t.Setenv(key, "")
	}

	if !appleAudienceMatches("unconfigured.app") {
		t.Fatal("Apple audience must be accepted when no client is configured")
	}
}

func TestAppleAudienceMatchesOnlyConfiguredClients(t *testing.T) {
	for _, key := range []string{"APPLE_CLIENT_ID", "APPLE_CLIENT_ID_IOS", "APPLE_CLIENT_ID_WEB", "APPLE_SERVICE_ID"} {
		t.Setenv(key, "")
	}

	t.Setenv("APPLE_CLIENT_ID_IOS", "com.example.ios")
	t.Setenv("APPLE_SERVICE_ID", "com.example.web")

	for _, audience := range []interface{}{"com.example.ios", "com.example.web", []interface{}{"other.app", "com.example.web"}} {
		if !appleAudienceMatches(audience) {
			t.Fatalf("configured audience %v was rejected", audience)
		}
	}
	for _, audience := range []interface{}{"other.app", "", nil, []interface{}{"other.app", 12}} {
		if appleAudienceMatches(audience) {
			t.Fatalf("unconfigured audience %v was accepted", audience)
		}
	}
}

func TestAppleAuthRejectsNonRS256Token(t *testing.T) {
	invalidToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"email": "user@example.com"})
	invalidToken.Header["kid"] = "example"
	signedToken, err := invalidToken.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := AppleAuth(signedToken, ""); err == nil {
		t.Fatal("a token signed with HS256 must be rejected")
	}
}

func TestAppleRSAPublicKeyRejectsInvalidExponent(t *testing.T) {
	modulus := base64.RawURLEncoding.EncodeToString([]byte{2})
	exponent := base64.RawURLEncoding.EncodeToString([]byte{0})
	if _, err := buildAppleRSAPublicKey(modulus, exponent); err == nil {
		t.Fatal("an invalid RSA exponent must be rejected")
	}
}
