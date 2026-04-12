package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sttts/xf-cli/auth"
	"github.com/sttts/xf-cli/scraper"
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
	client, err := auth.NewClient("https://www.rc-network.de", 0)
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

func TestLiveSessionRoundTrip(t *testing.T) {
	_, session := newLiveSession(t)
	path := filepath.Join(t.TempDir(), "session.json")

	if err := auth.SaveSession(path, session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := auth.LoadSession(path)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}

	reusedClient, err := auth.NewClient("https://www.rc-network.de", 0)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	verified, err := reusedClient.VerifySession(loaded)
	if err != nil {
		t.Fatalf("verify session: %v", err)
	}

	if verified.Username != session.Username {
		t.Fatalf("expected username %q, got %q", session.Username, verified.Username)
	}
	if verified.XFToken == "" {
		t.Fatal("expected refreshed xf token")
	}
	if len(verified.Cookies) == 0 {
		t.Fatal("expected cookies after verification")
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
	result, err := scraper.ListThreads(client, session, "/forums/flugmodellbau-allgemein.31/", "", 100)
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

func TestLiveListThreadsCursor(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListThreads(client, session, "/forums/flugmodellbau-allgemein.31/", "", 1)
	if err != nil {
		t.Fatalf("list threads with cursor: %v", err)
	}
	if result.NextPage == "" {
		t.Fatal("expected next page cursor")
	}

	nextResult, err := scraper.ListThreads(client, session, "/forums/flugmodellbau-allgemein.31/", result.NextPage, 1)
	if err != nil {
		t.Fatalf("list threads next cursor: %v", err)
	}
	if nextResult.Page <= result.Page {
		t.Fatalf("expected next page to advance, got %d after %d", nextResult.Page, result.Page)
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

func TestLiveSearchThreads(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.SearchThreads(client, session, "segler", "", 100)
	if err != nil {
		t.Fatalf("search threads: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected thread search results")
	}
	if result.SearchType != string(scraper.SearchModeThreads) {
		t.Fatalf("expected thread search type, got %q", result.SearchType)
	}
}

func TestLiveSearchThreadsCursor(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.SearchThreads(client, session, "segler", "", 1)
	if err != nil {
		t.Fatalf("search threads with cursor: %v", err)
	}
	if result.NextPage == "" {
		t.Fatal("expected next page cursor")
	}

	nextResult, err := scraper.SearchThreads(client, session, "segler", result.NextPage, 1)
	if err != nil {
		t.Fatalf("search threads next cursor: %v", err)
	}
	if nextResult.Page <= result.Page {
		t.Fatalf("expected next page to advance, got %d after %d", nextResult.Page, result.Page)
	}
}

func TestLiveSearchPosts(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.SearchPosts(client, session, "segler", "", 100)
	if err != nil {
		t.Fatalf("search posts: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected post search results")
	}
	if result.SearchType != string(scraper.SearchModePosts) {
		t.Fatalf("expected post search type, got %q", result.SearchType)
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
	result, err := scraper.ListThreads(client, session, "/find-threads/started", "", 100)
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
	result, err := scraper.ListThreads(client, session, "/find-threads/contributed", "", 100)
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

func TestLiveReadProfile(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ReadProfile(client, session, "/members/sttts.31018/")
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if result.DisplayName != "sttts" {
		t.Fatalf("expected display name sttts, got %q", result.DisplayName)
	}
	if result.RecentContentURL == "" {
		t.Fatal("expected recent content url")
	}
	if result.AllThreadsURL == "" {
		t.Fatal("expected all threads url")
	}
}

func TestLiveListUserPosts(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListUserPosts(client, session, "/members/sttts.31018/", "", 100)
	if err != nil {
		t.Fatalf("list user posts: %v", err)
	}
	if len(result.Posts) == 0 {
		t.Fatal("expected user posts")
	}
	if result.Posts[0].PostURL == "" {
		t.Fatal("expected post url")
	}
}

func TestLiveListUserThreads(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListUserThreads(client, session, "/members/sttts.31018/", "", 100)
	if err != nil {
		t.Fatalf("list user threads: %v", err)
	}
	if len(result.Threads) == 0 {
		t.Fatal("expected user threads")
	}
	if result.Threads[0].URL == "" {
		t.Fatal("expected thread url")
	}
}

func TestLiveListMyThreads(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListMyThreads(client, session, "", 100)
	if err != nil {
		t.Fatalf("list my threads: %v", err)
	}
	if len(result.Threads) == 0 {
		t.Fatal("expected my threads")
	}
}

func TestLiveListThreadsIParticipated(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.ListThreadsIParticipated(client, session, "", 100)
	if err != nil {
		t.Fatalf("list threads i participated: %v", err)
	}
	if len(result.Threads) == 0 {
		t.Fatal("expected participated threads")
	}
}

func TestLiveFollowLinkForum(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.FollowLink(client, session, "/forums/flugmodellbau-allgemein.31/")
	if err != nil {
		t.Fatalf("follow forum link: %v", err)
	}
	if result.Type != scraper.LinkTypeForum {
		t.Fatalf("expected forum type, got %s", result.Type)
	}
	if result.ForumURL == "" {
		t.Fatal("expected canonical forum url")
	}
}

func TestLiveFollowLinkThreadUnread(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.FollowLink(client, session, "/threads/eure-sch%C3%B6nsten-modelle.144946/unread")
	if err != nil {
		t.Fatalf("follow unread link: %v", err)
	}
	if result.Type != scraper.LinkTypeThread {
		t.Fatalf("expected thread type, got %s", result.Type)
	}
	if result.ThreadURL == "" || strings.Contains(result.ThreadURL, "/unread") {
		t.Fatalf("expected canonical thread url, got %q", result.ThreadURL)
	}
}

func TestLiveFollowLinkThreadLatest(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.FollowLink(client, session, "/threads/eure-sch%C3%B6nsten-modelle.144946/latest")
	if err != nil {
		t.Fatalf("follow latest link: %v", err)
	}
	if result.Type != scraper.LinkTypePost {
		t.Fatalf("expected post type, got %s", result.Type)
	}
	if result.PostURL == "" || !strings.Contains(result.PostURL, "#post-") && !strings.Contains(result.ResolvedURL, "#post-") {
		t.Fatalf("expected resolved post link, got post=%q resolved=%q", result.PostURL, result.ResolvedURL)
	}
}

func TestLiveFollowLinkAttachment(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.FollowLink(client, session, "/attachments/piper-tc-jpg.9277151/")
	if err != nil {
		t.Fatalf("follow attachment link: %v", err)
	}
	if result.Type != scraper.LinkTypeAttachment {
		t.Fatalf("expected attachment type, got %s", result.Type)
	}
	if result.AttachmentURL == "" {
		t.Fatal("expected attachment url")
	}
	if result.ImageURL == "" {
		t.Fatal("expected image url from attachment")
	}
}

func TestLiveGetImageFromAttachment(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.GetImage(client, session, "/attachments/piper-tc-jpg.9277151/")
	if err != nil {
		t.Fatalf("get image from attachment: %v", err)
	}
	if result.AttachmentURL == "" {
		t.Fatal("expected attachment url")
	}
	if result.FullImageURL == "" {
		t.Fatal("expected full image url")
	}
}

func TestLiveGetImageFromDirectImage(t *testing.T) {
	client, session := newLiveSession(t)
	result, err := scraper.GetImage(client, session, "https://www.rc-network.de/data/attachments/205/205458-62467cc69c476761618577652498d9f5.jpg?hash=YkZ8xpxHZ2")
	if err != nil {
		t.Fatalf("get image from direct image: %v", err)
	}
	if result.ThumbnailURL == "" {
		t.Fatal("expected thumbnail url")
	}
	if result.FullImageURL == "" {
		t.Fatal("expected full image url")
	}
}
