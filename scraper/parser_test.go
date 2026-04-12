package scraper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sttts/xf-cli/auth"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

func testClient(t *testing.T) *auth.Client {
	t.Helper()

	client, err := auth.NewClient("https://www.rc-network.de", 0)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	return client
}

func TestParseThreadListFixture(t *testing.T) {
	result, err := parseThreadList(testClient(t), fixture(t, "thread_list.html"), 1)
	if err != nil {
		t.Fatalf("parse thread list: %v", err)
	}
	if result.ForumTitle != "Flugmodellbau allgemein" {
		t.Fatalf("unexpected forum title %q", result.ForumTitle)
	}
	if len(result.Threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(result.Threads))
	}
	if result.NextPageURL == "" {
		t.Fatal("expected next page url")
	}
}

func TestParseThreadPostsFixture(t *testing.T) {
	title, posts, next, err := parseThreadPosts(testClient(t), fixture(t, "thread_posts.html"))
	if err != nil {
		t.Fatalf("parse thread posts: %v", err)
	}
	if title != "Test Thread" {
		t.Fatalf("unexpected title %q", title)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Author != "Alice" {
		t.Fatalf("unexpected author %q", posts[0].Author)
	}
	if len(posts[0].Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(posts[0].Images))
	}
	if next == "" {
		t.Fatal("expected next url")
	}
}

func TestParseSearchResultsFixture(t *testing.T) {
	result, err := parseSearchResults(testClient(t), fixture(t, "search_results.html"), "Fouga", 1)
	if err != nil {
		t.Fatalf("parse search results: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Title != "Fouga Aufbau" {
		t.Fatalf("unexpected title %q", result.Results[0].Title)
	}
	if result.NextPageURL == "" {
		t.Fatal("expected next page url")
	}
}

func TestParseUserProfileFixture(t *testing.T) {
	result, err := parseUserProfile(testClient(t), fixture(t, "user_profile.html"))
	if err != nil {
		t.Fatalf("parse user profile: %v", err)
	}
	if result.DisplayName != "sttts" {
		t.Fatalf("unexpected display name %q", result.DisplayName)
	}
	if result.RecentContentURL == "" || result.AllThreadsURL == "" {
		t.Fatal("expected profile urls")
	}
}

func TestPageCursorHelpers(t *testing.T) {
	page, err := parsePageCursor("2")
	if err != nil {
		t.Fatalf("parse page cursor: %v", err)
	}
	if page != 2 {
		t.Fatalf("expected page 2, got %d", page)
	}
	if nextCursorFromURL(2, "https://www.rc-network.de/page-3") != "3" {
		t.Fatal("expected next cursor 3")
	}
	if canonicalThreadURL("https://www.rc-network.de/threads/test.1/latest") != "https://www.rc-network.de/threads/test.1" {
		t.Fatal("expected canonical thread url without latest suffix")
	}
}
