package xfmcp

import "testing"

func TestConfigValidateAllowsStoredSessionWithoutCredentials(t *testing.T) {
	cfg := Config{BaseURL: "https://www.rc-network.de"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config without credentials to be valid, got %v", err)
	}
}

func TestConfigValidateRejectsPartialCredentials(t *testing.T) {
	tests := []Config{
		{BaseURL: "https://www.rc-network.de", Username: "user"},
		{BaseURL: "https://www.rc-network.de", Password: "pass"},
	}

	for _, cfg := range tests {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected partial credentials to be rejected for %+v", cfg)
		}
	}
}
