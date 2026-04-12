package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sttts/xf-cli/auth"
)

type UserProfile struct {
	Username         string `json:"username"`
	BaseURL          string `json:"base_url"`
	UserURL          string `json:"user_url"`
	DisplayName      string `json:"display_name"`
	UserTitle        string `json:"user_title,omitempty"`
	JoinedAt         string `json:"joined_at,omitempty"`
	LastActivity     string `json:"last_activity,omitempty"`
	PostCount        string `json:"post_count,omitempty"`
	ReactionScore    string `json:"reaction_score,omitempty"`
	RecentContentURL string `json:"recent_content_url,omitempty"`
	AboutURL         string `json:"about_url,omitempty"`
	AllContentURL    string `json:"all_content_url,omitempty"`
	AllThreadsURL    string `json:"all_threads_url,omitempty"`
}

type UserPostSummary struct {
	Title       string `json:"title"`
	PostURL     string `json:"post_url"`
	ThreadURL   string `json:"thread_url,omitempty"`
	ThreadTitle string `json:"thread_title,omitempty"`
	PostedAt    string `json:"posted_at,omitempty"`
	ForumTitle  string `json:"forum_title,omitempty"`
	ForumURL    string `json:"forum_url,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
}

type UserPostsResult struct {
	Username    string            `json:"username"`
	BaseURL     string            `json:"base_url"`
	UserURL     string            `json:"user_url"`
	Page        int               `json:"page"`
	Limit       int               `json:"limit,omitempty"`
	PagesRead   int               `json:"pages_read,omitempty"`
	Posts       []UserPostSummary `json:"posts"`
	NextPage    string            `json:"next_page,omitempty"`
	NextPageURL string            `json:"next_page_url,omitempty"`
}

type UserThreadSummary struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet,omitempty"`
	PostedAt string `json:"posted_at,omitempty"`
}

type UserThreadsResult struct {
	Username    string              `json:"username"`
	BaseURL     string              `json:"base_url"`
	UserURL     string              `json:"user_url"`
	Page        int                 `json:"page"`
	Limit       int                 `json:"limit,omitempty"`
	PagesRead   int                 `json:"pages_read,omitempty"`
	Threads     []UserThreadSummary `json:"threads"`
	NextPage    string              `json:"next_page,omitempty"`
	NextPageURL string              `json:"next_page_url,omitempty"`
}

func ReadProfile(client *auth.Client, session auth.SessionInfo, userRef string) (UserProfile, error) {
	userURL := client.ResolveURL(userRef)
	body, err := client.FetchPage(userURL)
	if err != nil {
		return UserProfile{}, fmt.Errorf("fetching profile: %w", err)
	}

	profile, err := parseUserProfile(client, body)
	if err != nil {
		return UserProfile{}, fmt.Errorf("parsing profile: %w", err)
	}

	profile.Username = session.Username
	profile.BaseURL = session.BaseURL
	if profile.UserURL == "" {
		profile.UserURL = userURL
	}
	return profile, nil
}

func ListUserPosts(client *auth.Client, session auth.SessionInfo, userRef string, cursor string, limit int) (UserPostsResult, error) {
	startPage, err := parsePageCursor(cursor)
	if err != nil {
		return UserPostsResult{}, err
	}
	limit = normalizeLimit(limit)

	profile, err := ReadProfile(client, session, userRef)
	if err != nil {
		return UserPostsResult{}, err
	}

	currentURL := profile.RecentContentURL
	if currentURL == "" {
		return UserPostsResult{}, fmt.Errorf("profile does not expose recent content URL")
	}

	var currentPage int
	var result UserPostsResult
	for currentPage = 1; currentPage <= startPage; currentPage++ {
		body, err := client.FetchPage(currentURL)
		if err != nil {
			return UserPostsResult{}, fmt.Errorf("fetching user posts page %d: %w", currentPage, err)
		}

		result, err = parseUserPosts(client, body, profile.UserURL, currentPage)
		if err != nil {
			return UserPostsResult{}, fmt.Errorf("parsing user posts page %d: %w", currentPage, err)
		}

		if currentPage < startPage {
			if result.NextPageURL == "" {
				break
			}
			currentURL = result.NextPageURL
		}
	}

	result.Page = currentPage - 1
	result.Limit = limit
	result.PagesRead = 1
	result.NextPage = nextCursorFromURL(result.Page, result.NextPageURL)
	for (limit == 0 || len(result.Posts) < limit) && result.NextPageURL != "" {
		body, err := client.FetchPage(result.NextPageURL)
		if err != nil {
			return UserPostsResult{}, fmt.Errorf("fetching user posts page %d: %w", result.Page+result.PagesRead, err)
		}

		pageResult, err := parseUserPosts(client, body, profile.UserURL, result.Page+result.PagesRead)
		if err != nil {
			return UserPostsResult{}, fmt.Errorf("parsing user posts page %d: %w", result.Page+result.PagesRead, err)
		}

		result.Posts = append(result.Posts, pageResult.Posts...)
		result.NextPageURL = pageResult.NextPageURL
		result.PagesRead++
		result.NextPage = nextCursorFromURL(result.Page+result.PagesRead-1, pageResult.NextPageURL)
	}

	result.Username = session.Username
	result.BaseURL = session.BaseURL
	result.UserURL = profile.UserURL
	return result, nil
}

func ListUserThreads(client *auth.Client, session auth.SessionInfo, userRef string, cursor string, limit int) (UserThreadsResult, error) {
	startPage, err := parsePageCursor(cursor)
	if err != nil {
		return UserThreadsResult{}, err
	}
	limit = normalizeLimit(limit)

	profile, err := ReadProfile(client, session, userRef)
	if err != nil {
		return UserThreadsResult{}, err
	}

	currentURL := profile.AllThreadsURL
	if currentURL == "" {
		return UserThreadsResult{}, fmt.Errorf("profile does not expose user threads URL")
	}

	var currentPage int
	var result UserThreadsResult
	for currentPage = 1; currentPage <= startPage; currentPage++ {
		body, err := client.FetchPage(currentURL)
		if err != nil {
			return UserThreadsResult{}, fmt.Errorf("fetching user threads page %d: %w", currentPage, err)
		}

		result, err = parseUserThreads(client, body, profile.UserURL, currentPage)
		if err != nil {
			return UserThreadsResult{}, fmt.Errorf("parsing user threads page %d: %w", currentPage, err)
		}

		if currentPage < startPage {
			if result.NextPageURL == "" {
				break
			}
			currentURL = result.NextPageURL
		}
	}

	result.Page = currentPage - 1
	result.Limit = limit
	result.PagesRead = 1
	result.NextPage = nextCursorFromURL(result.Page, result.NextPageURL)
	for (limit == 0 || len(result.Threads) < limit) && result.NextPageURL != "" {
		body, err := client.FetchPage(result.NextPageURL)
		if err != nil {
			return UserThreadsResult{}, fmt.Errorf("fetching user threads page %d: %w", result.Page+result.PagesRead, err)
		}

		pageResult, err := parseUserThreads(client, body, profile.UserURL, result.Page+result.PagesRead)
		if err != nil {
			return UserThreadsResult{}, fmt.Errorf("parsing user threads page %d: %w", result.Page+result.PagesRead, err)
		}

		result.Threads = append(result.Threads, pageResult.Threads...)
		result.NextPageURL = pageResult.NextPageURL
		result.PagesRead++
		result.NextPage = nextCursorFromURL(result.Page+result.PagesRead-1, pageResult.NextPageURL)
	}

	result.Username = session.Username
	result.BaseURL = session.BaseURL
	result.UserURL = profile.UserURL
	return result, nil
}

func ListMyThreads(client *auth.Client, session auth.SessionInfo, cursor string, limit int) (ThreadListResult, error) {
	return ListThreads(client, session, "/find-threads/started", cursor, limit)
}

func ListThreadsIParticipated(client *auth.Client, session auth.SessionInfo, cursor string, limit int) (ThreadListResult, error) {
	return ListThreads(client, session, "/find-threads/contributed", cursor, limit)
}

func parseUserProfile(client *auth.Client, body []byte) (UserProfile, error) {
	doc, err := newDocument(body)
	if err != nil {
		return UserProfile{}, err
	}

	profile := UserProfile{}
	profile.DisplayName = cleanText(doc.Find(".memberHeader-name .username").First().Text())
	profile.UserTitle = cleanText(doc.Find(".memberHeader-blurb .userTitle").First().Text())
	profile.JoinedAt = cleanText(doc.Find(".memberHeader-blurb dt").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return cleanText(s.Text()) == "Registriert"
	}).Parent().Find("dd time").First().AttrOr("title", ""))
	profile.LastActivity = cleanText(doc.Find(".memberHeader-blurb dt").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return cleanText(s.Text()) == "Letzte Aktivität"
	}).Parent().Find("dd").First().Text())
	profile.PostCount = cleanText(doc.Find(".pairs dt").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return cleanText(s.Text()) == "Beiträge"
	}).Parent().Find("dd").First().Text())
	profile.ReactionScore = cleanText(doc.Find(".pairs dt").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return cleanText(s.Text()) == "Reaktionspunkte"
	}).Parent().Find("dd").First().Text())

	if href, ok := doc.Find(`link[rel="canonical"]`).Attr("href"); ok {
		profile.UserURL = client.ResolveURL(href)
	}
	if href, ok := doc.Find(`a[href*="/recent-content"]`).First().Attr("href"); ok {
		profile.RecentContentURL = client.ResolveURL(href)
	}
	if href, ok := doc.Find(`a[href*="/about"]`).First().Attr("href"); ok {
		profile.AboutURL = client.ResolveURL(href)
	}
	doc.Find(".menu-linkRow").Each(func(_ int, sel *goquery.Selection) {
		text := cleanText(sel.Text())
		href := client.ResolveURL(attrOrEmpty(sel, "href"))
		switch {
		case strings.Contains(text, "Finde alle Inhalte"):
			profile.AllContentURL = href
		case strings.Contains(text, "Finde alle Themen"):
			profile.AllThreadsURL = href
		}
	})

	return profile, nil
}

func parseUserPosts(client *auth.Client, body []byte, userURL string, page int) (UserPostsResult, error) {
	doc, err := newDocument(body)
	if err != nil {
		return UserPostsResult{}, err
	}

	result := UserPostsResult{
		UserURL:     userURL,
		Page:        page,
		Posts:       make([]UserPostSummary, 0),
		NextPageURL: nextPageURL(client, doc),
	}

	doc.Find(".contentRow").Each(func(_ int, row *goquery.Selection) {
		titleLink := row.Find(".contentRow-title a").First()
		href := client.ResolveURL(attrOrEmpty(titleLink, "href"))
		if href == "" || strings.Contains(href, "/direct-messages/") || !strings.Contains(href, "/post-") {
			return
		}

		title := cleanText(titleLink.Text())
		if title == "" {
			return
		}

		post := UserPostSummary{
			Title:    title,
			PostURL:  href,
			Snippet:  cleanText(row.Find(".contentRow-snippet").First().Text()),
			PostedAt: cleanText(row.Find(".contentRow-minor time").First().AttrOr("title", row.Find(".contentRow-minor time").First().Text())),
		}

		if idx := strings.Index(post.PostURL, "/post-"); idx > 0 {
			post.ThreadURL = post.PostURL[:idx]
			post.ThreadTitle = title
		}

		row.Find(".contentRow-minor a").Each(func(_ int, link *goquery.Selection) {
			linkHref := client.ResolveURL(attrOrEmpty(link, "href"))
			if strings.Contains(linkHref, "/forums/") {
				post.ForumURL = linkHref
				post.ForumTitle = cleanText(link.Text())
			}
		})

		result.Posts = append(result.Posts, post)
	})

	return result, nil
}

func parseUserThreads(client *auth.Client, body []byte, userURL string, page int) (UserThreadsResult, error) {
	doc, err := newDocument(body)
	if err != nil {
		return UserThreadsResult{}, err
	}

	result := UserThreadsResult{
		UserURL:     userURL,
		Page:        page,
		Threads:     make([]UserThreadSummary, 0),
		NextPageURL: nextPageURL(client, doc),
	}

	doc.Find(".contentRow").Each(func(_ int, row *goquery.Selection) {
		titleLink := row.Find(".contentRow-title a").First()
		href := client.ResolveURL(attrOrEmpty(titleLink, "href"))
		if href == "" || strings.Contains(href, "/direct-messages/") || strings.Contains(href, "/post-") {
			return
		}

		title := cleanText(titleLink.Text())
		if title == "" {
			return
		}

		thread := UserThreadSummary{
			Title:    title,
			URL:      href,
			Snippet:  cleanText(row.Find(".contentRow-snippet").First().Text()),
			PostedAt: cleanText(row.Find(".contentRow-minor time").First().AttrOr("title", row.Find(".contentRow-minor time").First().Text())),
		}

		result.Threads = append(result.Threads, thread)
	})

	return result, nil
}
