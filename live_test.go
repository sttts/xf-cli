package main

import (
	"os"
	"strings"
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

	foundImage := false
	for _, post := range result.Posts {
		for _, image := range post.Images {
			if image.URL != "" {
				foundImage = true
				break
			}
		}
		if foundImage {
			break
		}
	}
	if !foundImage {
		t.Fatal("expected extracted images in thread")
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

func TestLiveProfilePage(t *testing.T) {
	client, _ := newLiveSession(t)
	body, err := client.FetchPage(client.BaseURL() + "/members/sttts.31018/")
	if err != nil {
		t.Fatalf("fetch profile page: %v", err)
	}

	page := string(body)
	if !strings.Contains(page, "memberHeader-name") {
		t.Fatal("expected member header on profile page")
	}
	if !strings.Contains(page, "/members/sttts.31018/recent-content") {
		t.Fatal("expected recent-content link on profile page")
	}
}

func TestLiveRecentContentPage(t *testing.T) {
	client, _ := newLiveSession(t)
	body, err := client.FetchPage(client.BaseURL() + "/members/sttts.31018/recent-content")
	if err != nil {
		t.Fatalf("fetch recent content page: %v", err)
	}

	page := string(body)
	if !strings.Contains(page, "Aktueller Inhalt von sttts") {
		t.Fatal("expected recent content title")
	}
	if !strings.Contains(page, "contentRow-title") {
		t.Fatal("expected content rows on recent content page")
	}
}

func TestLiveFindStartedThreads(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListThreads(client, session, "/find-threads/started", 1)
	if err != nil {
		t.Fatalf("list started threads: %v", err)
	}
	if !strings.Contains(result.ForumTitle, "Themen") {
		t.Fatalf("expected started threads title, got %q", result.ForumTitle)
	}
	if len(result.Threads) == 0 {
		t.Fatal("expected started threads")
	}
}

func TestLiveFindContributedThreads(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListThreads(client, session, "/find-threads/contributed", 1)
	if err != nil {
		t.Fatalf("list contributed threads: %v", err)
	}
	if !strings.Contains(result.ForumTitle, "Themen") {
		t.Fatalf("expected contributed threads title, got %q", result.ForumTitle)
	}
	if len(result.Threads) == 0 {
		t.Fatal("expected contributed threads")
	}
}
