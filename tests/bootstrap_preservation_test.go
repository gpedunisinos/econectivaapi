package tests

import (
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/stretchr/testify/require"
)

func TestBootstrapPreservesExistingAdminAccount(t *testing.T) {
	setupDatabase()
	originalDB := config.DB
	tx := originalDB.Begin()
	require.NoError(t, tx.Error)
	defer func() { tx.Rollback(); config.DB = originalDB }()
	config.DB = tx
	var admin models.User
	require.NoError(t, tx.Where("email = ?", "admin@admin.com").First(&admin).Error)
	mode := "inventor"
	require.NoError(t, tx.Model(&admin).Updates(map[string]interface{}{"name": "Nome mantido na reinicialização", "password": "synthetic-existing-hash", "profile": 10, "usage_mode": mode}).Error)
	config.CreateOrUpdateAdminAccount()
	var after models.User
	require.NoError(t, tx.First(&after, admin.ID).Error)
	require.Equal(t, "Nome mantido na reinicialização", after.Name)
	require.Equal(t, "synthetic-existing-hash", after.Password)
	require.EqualValues(t, 10, after.Profile)
	require.Equal(t, &mode, after.UsageMode)
	require.NoError(t, tx.Delete(&after).Error)
	config.CreateOrUpdateAdminAccount() // Must not resurrect a disabled administrator.
	var count int64
	require.NoError(t, tx.Model(&models.User{}).Where("email = ?", admin.Email).Count(&count).Error)
	require.Zero(t, count)
}
