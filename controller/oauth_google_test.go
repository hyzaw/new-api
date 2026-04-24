package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/stretchr/testify/require"
)

func setupOAuthGoogleTestDB(t *testing.T) {
	t.Helper()

	db := openTokenControllerTestDB(t)
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate user table: %v", err)
	}
}

func TestTryAutoBindOAuthUserByEmailForGoogle(t *testing.T) {
	setupOAuthGoogleTestDB(t)
	common.RedisEnabled = false

	existingUser := &model.User{
		Username: "existing_google_email",
		Password: "dummy-password",
		Email:    "alice@gmail.com",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
	}
	require.NoError(t, model.DB.Create(existingUser).Error)

	provider := &oauth.GoogleProvider{}
	oauthUser := &oauth.OAuthUser{
		ProviderUserID: "google-sub-123",
		Email:          "alice@gmail.com",
		Extra: map[string]any{
			"email_verified": true,
		},
	}

	user, err := tryAutoBindOAuthUserByEmail(provider, oauthUser)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, existingUser.Id, user.Id)
	require.Equal(t, "google-sub-123", user.GoogleId)

	var reloaded model.User
	require.NoError(t, model.DB.Where("id = ?", existingUser.Id).First(&reloaded).Error)
	require.Equal(t, "google-sub-123", reloaded.GoogleId)
	require.Equal(t, "alice@gmail.com", reloaded.Email)
}

func TestSyncOAuthEmailForExistingUserWhenBindingGoogle(t *testing.T) {
	setupOAuthGoogleTestDB(t)
	common.RedisEnabled = false

	user := &model.User{
		Username: "bind_google_email",
		Password: "dummy-password",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
	}
	require.NoError(t, model.DB.Create(user).Error)

	provider := &oauth.GoogleProvider{}
	oauthUser := &oauth.OAuthUser{
		ProviderUserID: "google-sub-456",
		Email:          "bind@gmail.com",
		Extra: map[string]any{
			"email_verified": true,
		},
	}

	require.NoError(t, syncOAuthEmailForExistingUser(user, oauthUser, provider))

	var reloaded model.User
	require.NoError(t, model.DB.Where("id = ?", user.Id).First(&reloaded).Error)
	require.Equal(t, "bind@gmail.com", reloaded.Email)
}
