package tests

import (
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseConnectionUsesUTF8MB4(t *testing.T) {
	config.InitDatabase()

	var charset string
	result := config.DB.Raw("SELECT @@character_set_connection").Scan(&charset)

	assert.NoError(t, result.Error)
	assert.Equal(t, "utf8mb4", charset)
}
