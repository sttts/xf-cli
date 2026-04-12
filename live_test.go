package main

import (
	"os"
	"testing"

	"github.com/sttts/xf-mcp/auth"
	"github.com/sttts/xf-mcp/scraper"
)

func requireLiveCredentials(t *testing.T) (string, string) {
	t.Helper()

	username := os.Getenv("XF_USERNAME")
	password := os.Getenv("XF_PASSWORD")
	if username == "" || password == "" {
		t.Skip("live test requires XF_USERNAME and XF_PASSWORD")
	}

	return username, password
}

func newLiveSession(t *testing.T) (*auth.Client, auth.SessionInfo) {
	t.Helper()

	username, password := requireLiveCredentials(t)
	client, err := auth.NewClient("https://www.rc-network.de", false)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	session, err := client.Login(username, password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	return client, session
}

func TestLiveLogin(t *testing.T) {
	_, session := newLiveSession(t)
	if session.Username == "" {
		t.Fatal("expected username in session")
	}
	if session.XFToken == "" {
		t.Fatal("expected xf token in session")
	}
	if len(session.Cookies) == 0 {
		t.Fatal("expected cookies in session")
	}
}

func TestLiveListForums(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListForums(client, session)
	if err != nil {
		t.Fatalf("list forums: %v", err)
	}
	if len(result.Categories) == 0 {
		t.Fatal("expected forum categories")
	}
}

func TestLiveListThreads(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListThreads(client, session, "/forums/flugmodellbau-allgemein.31/", 1)
	if err != nil {
		t.Fatalf("list threads: %v", err)
	}
	if result.ForumTitle == "" {
		t.Fatal("expected forum title")
	}
	if len(result.Threads) == 0 {
		t.Fatal("expected thread summaries")
	}
}

func TestLiveReadThread(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ReadThread(client, session, "/threads/eure-sch%C3%B6nsten-modelle.144946/")
	if err != nil {
		t.Fatalf("read thread: %v", err)
	}
	if result.Title == "" {
		t.Fatal("expected thread title")
	}
	if result.PagesRead < 2 {
		t.Fatalf("expected multiple pages, got %d", result.PagesRead)
	}
	if len(result.Posts) == 0 {
		t.Fatal("expected posts")
	}
}

func TestLiveSearch(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.Search(client, session, "segler", 1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected search results")
	}
}
