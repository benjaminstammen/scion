package cmd

import (
	"testing"

	"github.com/ptone/scion-agent/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestGetAuthInfo_NoAuth(t *testing.T) {
	settings := &config.Settings{}
	info := getAuthInfo(settings, "https://hub.example.com")
	assert.Equal(t, "none", info.MethodType)
	assert.Equal(t, "none", info.Method)
}

func TestGetAuthInfo_BearerToken(t *testing.T) {
	settings := &config.Settings{
		Hub: &config.HubClientConfig{
			Token: "test-token",
		},
	}
	info := getAuthInfo(settings, "https://hub.example.com")
	assert.Equal(t, "bearer", info.MethodType)
	assert.Equal(t, "settings", info.Source)
}

func TestGetAuthInfo_APIKey(t *testing.T) {
	settings := &config.Settings{
		Hub: &config.HubClientConfig{
			APIKey: "test-api-key",
		},
	}
	info := getAuthInfo(settings, "https://hub.example.com")
	assert.Equal(t, "apikey", info.MethodType)
	assert.Equal(t, "settings", info.Source)
}

func TestGetAuthInfo_BearerPrecedence(t *testing.T) {
	// Bearer token should take precedence over API key
	settings := &config.Settings{
		Hub: &config.HubClientConfig{
			Token:  "test-token",
			APIKey: "test-api-key",
		},
	}
	info := getAuthInfo(settings, "https://hub.example.com")
	assert.Equal(t, "bearer", info.MethodType)
}

func TestGetAuthInfo_NilHub(t *testing.T) {
	settings := &config.Settings{
		Hub: nil,
	}
	info := getAuthInfo(settings, "")
	assert.Equal(t, "none", info.MethodType)
}
