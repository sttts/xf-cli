package auth

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	session := SessionInfo{
		Username: "tester",
		Cookies: map[string]string{
			"xf2_session": "abc",
			"xf2_user":    "123",
		},
		XFToken: "token",
		BaseURL: "https://www.rc-network.de",
	}

	if err := SaveSession(path, session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := LoadSession(path)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}

	if loaded.Username != session.Username {
		t.Fatalf("expected username %q, got %q", session.Username, loaded.Username)
	}
	if loaded.XFToken != session.XFToken {
		t.Fatalf("expected token %q, got %q", session.XFToken, loaded.XFToken)
	}
	if loaded.Cookies["xf2_session"] != "abc" {
		t.Fatalf("expected session cookie to round-trip")
	}
}

func TestApplySessionSetsCookies(t *testing.T) {
	client, err := NewClient("https://www.rc-network.de", 0)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	session := SessionInfo{
		BaseURL: "https://www.rc-network.de",
		Cookies: map[string]string{
			"xf2_session": "abc",
			"xf2_user":    "123",
		},
	}

	if err := client.ApplySession(session); err != nil {
		t.Fatalf("apply session: %v", err)
	}

	cookies := client.Cookies()
	if cookies["xf2_session"] != "abc" {
		t.Fatalf("expected session cookie to be present")
	}
	if cookies["xf2_user"] != "123" {
		t.Fatalf("expected user cookie to be present")
	}
}
