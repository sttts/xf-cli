package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

func TestExtractCSRFTokenPrefersInput(t *testing.T) {
	token, err := ExtractCSRFToken(readFixture(t, "login_page.html"))
	if err != nil {
		t.Fatalf("extract csrf token: %v", err)
	}
	if token != "token-from-input" {
		t.Fatalf("expected token-from-input, got %q", token)
	}
}

func TestDetectLoginErrorFromBlockMessage(t *testing.T) {
	msg := DetectLoginError(readFixture(t, "login_error.html"))
	if msg == "" {
		t.Fatal("expected login error")
	}
}
