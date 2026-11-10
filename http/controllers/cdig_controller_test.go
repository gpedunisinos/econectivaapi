package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCDIGStatusNeedsCompleteConfigurationAndNeverExposesSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for key, value := range map[string]string{"CDIG_SYNC_ENABLED": "true", "CDIG_BASE_URL": "https://cdig.example", "CDIG_SOURCE_KEY": "test", "CDIG_IMPORT_USER_ID": "1", "CDIG_API_TOKEN": "synthetic-private-token", "CDIG_EMAIL": "", "CDIG_PASSWORD": ""} {
		t.Setenv(key, value)
	}
	request := func() string {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		CDIGIntegrationStatus(c)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
		return w.Body.String()
	}
	response := request()
	if !strings.Contains(response, `"enabled":true`) || strings.Contains(response, "synthetic-private-token") {
		t.Fatal(response)
	}
	t.Setenv("CDIG_API_TOKEN", "")
	response = request()
	if !strings.Contains(response, `"enabled":false`) || strings.Contains(response, "https://") {
		t.Fatal(response)
	}
	t.Setenv("CDIG_SYNC_ENABLED", "false")
	response = request()
	if !strings.Contains(response, `"enabled":false`) {
		t.Fatal(response)
	}
}
