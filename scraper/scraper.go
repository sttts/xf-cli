package scraper

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sttts/xf-cli/auth"
)

type ForumCategory struct {
	Title       string      `json:"title"`
	URL         string      `json:"url"`
	Description string      `json:"description,omitempty"`
	Forums      []ForumNode `json:"forums"`
}

type ForumNode struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type ForumListResult struct {
	Username   string          `json:"username"`
	BaseURL    string          `json:"base_url"`
	Categories []ForumCategory `json:"categories"`
}

type ThreadSummary struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	Author     string `json:"author,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	Replies    string `json:"replies,omitempty"`
	Views      string `json:"views,omitempty"`
	LastPostAt string `json:"last_post_at,omitempty"`
	LastPoster string `json:"last_poster,omitempty"`
}

type ThreadListResult struct {
	Username    string          `json:"username"`
	BaseURL     string          `json:"base_url"`
	ForumTitle  string          `json:"forum_title"`
	ForumURL    string          `json:"forum_url"`
	Page        int             `json:"page"`
	Limit       int             `json:"limit,omitempty"`
	PagesRead   int             `json:"pages_read,omitempty"`
	Threads     []ThreadSummary `json:"threads"`
	NextPage    string          `json:"next_page,omitempty"`
	NextPageURL string          `json:"next_page_url,omitempty"`
}

type PostImage struct {
	URL           string `json:"url"`
	PreviewURL    string `json:"preview_url,omitempty"`
	Alt           string `json:"alt,omitempty"`
	AttachmentURL string `json:"attachment_url,omitempty"`
}

type ThreadPost struct {
	PostNumber string      `json:"post_number,omitempty"`
	PostURL    string      `json:"post_url,omitempty"`
	Author     string      `json:"author"`
	PostedAt   string      `json:"posted_at,omitempty"`
	Content    string      `json:"content"`
	Images     []PostImage `json:"images,omitempty"`
}

type ThreadReadResult struct {
	Username  string       `json:"username"`
	BaseURL   string       `json:"base_url"`
	ThreadURL string       `json:"thread_url"`
	Title     string       `json:"title"`
	Posts     []ThreadPost `json:"posts"`
	PagesRead int          `json:"pages_read"`
}

type SearchResultItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

type SearchResult struct {
	Username    string             `json:"username"`
	BaseURL     string             `json:"base_url"`
	SearchType  string             `json:"search_type"`
	Query       string             `json:"query"`
	Page        int                `json:"page"`
	Limit       int                `json:"limit,omitempty"`
	PagesRead   int                `json:"pages_read,omitempty"`
	Results     []SearchResultItem `json:"results"`
	NextPage    string             `json:"next_page,omitempty"`
	NextPageURL string             `json:"next_page_url,omitempty"`
}

type SearchMode string

const (
	SearchModeThreads SearchMode = "threads"
	SearchModePosts   SearchMode = "posts"
)

func ListForums(client *auth.Client, session auth.SessionInfo) (ForumListResult, error) {
	body, err := client.FetchPage(client.BaseURL() + "/forums/")
	if err != nil {
		return ForumListResult{}, fmt.Errorf("fetching forum list: %w", err)
	}

	categories, err := parseForumList(client, body)
	if err != nil {
		return ForumListResult{}, fmt.Errorf("parsing forum list: %w", err)
	}

	return ForumListResult{
		Username:   session.Username,
		BaseURL:    session.BaseURL,
		Categories: categories,
	}, nil
}

func ListThreads(client *auth.Client, session auth.SessionInfo, forumRef string, cursor string, limit int) (ThreadListResult, error) {
	startPage, err := parsePageCursor(cursor)
	if err != nil {
		return ThreadListResult{}, err
	}
	limit = normalizeLimit(limit)

	result, err := collectThreadPages(client, forumRef, startPage, limit)
	if err != nil {
		return ThreadListResult{}, err
	}

	result.Username = session.Username
	result.BaseURL = session.BaseURL
	return result, nil
}

func listThreadsPage(client *auth.Client, forumRef string, page int) (ThreadListResult, error) {
	page = normalizePage(page)
	forumRef = forumURL(client, forumRef, page)

	body, err := client.FetchPage(forumRef)
	if err != nil {
		return ThreadListResult{}, fmt.Errorf("fetching thread list: %w", err)
	}

	result, err := parseThreadList(client, body, page)
	if err != nil {
		return ThreadListResult{}, fmt.Errorf("parsing thread list: %w", err)
	}

	result.ForumURL = forumRef
	return result, nil
}

func ReadThread(client *auth.Client, session auth.SessionInfo, threadRef string) (ThreadReadResult, error) {
	currentURL := threadURL(client, threadRef, 1)
	seen := make(map[string]struct{})
	posts := make([]ThreadPost, 0)
	title := ""
	pagesRead := 0

	for currentURL != "" {
		if _, ok := seen[currentURL]; ok {
			break
		}
		seen[currentURL] = struct{}{}

		body, err := client.FetchPage(currentURL)
		if err != nil {
			return ThreadReadResult{}, fmt.Errorf("fetching thread page %s: %w", currentURL, err)
		}

		pageTitle, pagePosts, nextURL, err := parseThreadPosts(client, body)
		if err != nil {
			return ThreadReadResult{}, fmt.Errorf("parsing thread page %s: %w", currentURL, err)
		}

		if title == "" {
			title = pageTitle
		}
		posts = append(posts, pagePosts...)
		pagesRead++
		currentURL = nextURL
	}

	return ThreadReadResult{
		Username:  session.Username,
		BaseURL:   session.BaseURL,
		ThreadURL: client.ResolveURL(threadRef),
		Title:     title,
		Posts:     posts,
		PagesRead: pagesRead,
	}, nil
}

func SearchThreads(client *auth.Client, session auth.SessionInfo, query string, cursor string, limit int) (SearchResult, error) {
	return search(client, session, query, cursor, limit, SearchModeThreads)
}

func SearchPosts(client *auth.Client, session auth.SessionInfo, query string, cursor string, limit int) (SearchResult, error) {
	return search(client, session, query, cursor, limit, SearchModePosts)
}

func search(client *auth.Client, session auth.SessionInfo, query string, cursor string, limit int, mode SearchMode) (SearchResult, error) {
	startPage, err := parsePageCursor(cursor)
	if err != nil {
		return SearchResult{}, err
	}
	limit = normalizeLimit(limit)

	searchPageURL := client.BaseURL() + "/search/"
	searchPageBody, err := client.FetchPage(searchPageURL)
	if err != nil {
		return SearchResult{}, fmt.Errorf("fetching search page: %w", err)
	}

	token, err := auth.ExtractCSRFToken(searchPageBody)
	if err != nil {
		return SearchResult{}, fmt.Errorf("extracting search token: %w", err)
	}

	form := url.Values{
		"keywords": {query},
		"_xfToken": {token},
	}
	switch mode {
	case SearchModeThreads:
		form.Set("c[title_only]", "1")
	case SearchModePosts:
		form.Set("search_type", "post")
	default:
		return SearchResult{}, fmt.Errorf("unsupported search mode %q", mode)
	}

	searchBody, err := client.PostForm(client.BaseURL()+"/search/search", form, searchPageURL)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search request failed: %w", err)
	}

	initialResult, err := parseSearchResults(client, searchBody, query, 1)
	if err != nil {
		return SearchResult{}, fmt.Errorf("parsing search results: %w", err)
	}

	result, err := collectSearchPages(client, query, mode, startPage, limit, initialResult)
	if err != nil {
		return SearchResult{}, err
	}

	result.Username = session.Username
	result.BaseURL = session.BaseURL
	result.SearchType = string(mode)
	return result, nil
}

func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizeLimit(limit int) int {
	if limit < 0 {
		return 100
	}
	return limit
}

func parsePageCursor(cursor string) (int, error) {
	if strings.TrimSpace(cursor) == "" {
		return 1, nil
	}

	page, err := strconv.Atoi(strings.TrimSpace(cursor))
	if err != nil || page < 1 {
		return 0, fmt.Errorf("invalid page cursor %q", cursor)
	}

	return page, nil
}

func formatPageCursor(page int) string {
	if page < 1 {
		return ""
	}
	return strconv.Itoa(page)
}

func collectThreadPages(client *auth.Client, forumRef string, startPage int, limit int) (ThreadListResult, error) {
	current, err := listThreadsPage(client, forumRef, 1)
	if err != nil {
		return ThreadListResult{}, err
	}

	currentPage := 1
	for currentPage < startPage {
		if current.NextPageURL == "" {
			break
		}

		body, err := client.FetchPage(current.NextPageURL)
		if err != nil {
			return ThreadListResult{}, fmt.Errorf("fetching thread list page %d: %w", currentPage+1, err)
		}

		current, err = parseThreadList(client, body, currentPage+1)
		if err != nil {
			return ThreadListResult{}, fmt.Errorf("parsing thread list page %d: %w", currentPage+1, err)
		}
		current.ForumURL = client.ResolveURL(forumRef)
		currentPage++
	}

	combined := current
	combined.Page = currentPage
	combined.Limit = limit
	combined.PagesRead = 1
	combined.NextPage = nextCursorFromURL(currentPage, combined.NextPageURL)

	for (limit == 0 || len(combined.Threads) < limit) && combined.NextPageURL != "" {
		body, err := client.FetchPage(combined.NextPageURL)
		if err != nil {
			return ThreadListResult{}, fmt.Errorf("fetching thread list page %d: %w", currentPage+1, err)
		}

		pageResult, err := parseThreadList(client, body, currentPage+1)
		if err != nil {
			return ThreadListResult{}, fmt.Errorf("parsing thread list page %d: %w", currentPage+1, err)
		}

		combined.Threads = append(combined.Threads, pageResult.Threads...)
		combined.NextPageURL = pageResult.NextPageURL
		currentPage++
		combined.NextPage = nextCursorFromURL(currentPage, pageResult.NextPageURL)
		combined.PagesRead++
	}

	return combined, nil
}

func collectSearchPages(client *auth.Client, query string, mode SearchMode, startPage int, limit int, initial SearchResult) (SearchResult, error) {
	current := initial
	currentPage := 1

	for currentPage < startPage {
		if current.NextPageURL == "" {
			break
		}

		body, err := client.FetchPage(current.NextPageURL)
		if err != nil {
			return SearchResult{}, fmt.Errorf("fetching search page %d: %w", currentPage+1, err)
		}

		current, err = parseSearchResults(client, body, query, currentPage+1)
		if err != nil {
			return SearchResult{}, fmt.Errorf("parsing search page %d: %w", currentPage+1, err)
		}

		currentPage++
	}

	combined := current
	combined.Page = currentPage
	combined.Limit = limit
	combined.PagesRead = 1
	combined.SearchType = string(mode)
	combined.NextPage = nextCursorFromURL(currentPage, combined.NextPageURL)

	for (limit == 0 || len(combined.Results) < limit) && combined.NextPageURL != "" {
		body, err := client.FetchPage(combined.NextPageURL)
		if err != nil {
			return SearchResult{}, fmt.Errorf("fetching search page %d: %w", currentPage+1, err)
		}

		pageResult, err := parseSearchResults(client, body, query, currentPage+1)
		if err != nil {
			return SearchResult{}, fmt.Errorf("parsing search page %d: %w", currentPage+1, err)
		}

		combined.Results = append(combined.Results, pageResult.Results...)
		combined.NextPageURL = pageResult.NextPageURL
		currentPage++
		combined.PagesRead++
		combined.NextPage = nextCursorFromURL(currentPage, pageResult.NextPageURL)
	}

	return combined, nil
}

func nextCursorFromURL(currentPage int, nextURL string) string {
	if nextURL == "" {
		return ""
	}

	return formatPageCursor(currentPage + 1)
}

func forumURL(client *auth.Client, forumRef string, page int) string {
	u := client.ResolveURL(forumRef)
	if page <= 1 {
		return u
	}
	return strings.TrimRight(u, "/") + fmt.Sprintf("/page-%d", page)
}

func threadURL(client *auth.Client, threadRef string, page int) string {
	u := client.ResolveURL(threadRef)
	if page <= 1 {
		return u
	}
	return strings.TrimRight(u, "/") + fmt.Sprintf("/page-%d", page)
}

func cleanText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func newDocument(body []byte) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	return doc, nil
}

func attrOrEmpty(sel *goquery.Selection, attr string) string {
	value, _ := sel.Attr(attr)
	return value
}

func extractTitle(doc *goquery.Document) string {
	return cleanText(doc.Find(".p-title-value").First().Text())
}

func nextPageURL(client *auth.Client, doc *goquery.Document) string {
	if href, ok := doc.Find(".pageNav-jump--next").First().Attr("href"); ok && href != "" {
		return client.ResolveURL(href)
	}

	if href, ok := doc.Find(".pageNavSimple-el--next").First().Attr("href"); ok && href != "" {
		return client.ResolveURL(href)
	}

	return ""
}

func parseForumList(client *auth.Client, body []byte) ([]ForumCategory, error) {
	doc, err := newDocument(body)
	if err != nil {
		return nil, err
	}

	categories := make([]ForumCategory, 0)
	doc.Find(".node.node--category").Each(func(_ int, categorySel *goquery.Selection) {
		category := ForumCategory{
			Forums: make([]ForumNode, 0),
		}

		if titleLink := categorySel.Find(".node-title a").First(); titleLink.Length() > 0 {
			category.Title = strings.TrimSpace(titleLink.Text())
			if href, ok := titleLink.Attr("href"); ok {
				category.URL = client.ResolveURL(href)
			}
		}

		category.Description = strings.TrimSpace(categorySel.Find(".node-description").First().Text())

		categorySel.Find(".subNodeLink--forum").Each(func(_ int, forumSel *goquery.Selection) {
			title := strings.TrimSpace(forumSel.Text())
			if title == "" {
				return
			}

			forum := ForumNode{Title: title}
			if href, ok := forumSel.Attr("href"); ok {
				forum.URL = client.ResolveURL(href)
			}
			category.Forums = append(category.Forums, forum)
		})

		if category.Title != "" && len(category.Forums) > 0 {
			categories = append(categories, category)
		}
	})

	return categories, nil
}

func parseThreadList(client *auth.Client, body []byte, page int) (ThreadListResult, error) {
	doc, err := newDocument(body)
	if err != nil {
		return ThreadListResult{}, err
	}

	result := ThreadListResult{
		ForumTitle:  extractTitle(doc),
		Page:        page,
		Threads:     make([]ThreadSummary, 0),
		NextPageURL: nextPageURL(client, doc),
	}

	doc.Find(".structItem--thread").Each(func(_ int, item *goquery.Selection) {
		titleLink := item.Find(".structItem-title a").First()
		title := cleanText(titleLink.Text())
		if title == "" {
			return
		}

		thread := ThreadSummary{
			Title: title,
			URL:   client.ResolveURL(attrOrEmpty(titleLink, "href")),
		}
		thread.Author = cleanText(item.Find(".structItem-minor .username").First().Text())
		thread.StartedAt = cleanText(item.Find(".structItem-startDate time").First().AttrOr("title", item.Find(".structItem-startDate time").First().Text()))
		thread.Replies = cleanText(item.Find(".structItem-cell--meta dd").First().Text())
		thread.Views = cleanText(item.Find(".structItem-cell--meta .structItem-minor dd").First().Text())
		thread.LastPostAt = cleanText(item.Find(".structItem-cell--latest time").First().AttrOr("title", item.Find(".structItem-cell--latest time").First().Text()))
		thread.LastPoster = cleanText(item.Find(".structItem-cell--latest .username").First().Text())
		result.Threads = append(result.Threads, thread)
	})

	return result, nil
}

func extractPostImages(client *auth.Client, post *goquery.Selection) []PostImage {
	images := make([]PostImage, 0)
	seen := make(map[string]struct{})

	post.Find(".message-userContent img").Each(func(_ int, imgSel *goquery.Selection) {
		className := imgSel.AttrOr("class", "")
		if strings.Contains(className, "smilie") || strings.Contains(className, "avatar") || strings.Contains(className, "reaction-image") {
			return
		}

		src := client.ResolveURL(attrOrEmpty(imgSel, "src"))
		if src == "" {
			return
		}
		if _, ok := seen[src]; ok {
			return
		}
		seen[src] = struct{}{}

		image := PostImage{
			URL:        src,
			PreviewURL: src,
			Alt:        cleanText(imgSel.AttrOr("alt", "")),
		}

		if link := imgSel.ParentFiltered("a"); link.Length() > 0 {
			image.AttachmentURL = client.ResolveURL(attrOrEmpty(link, "href"))
		} else if wrapper := imgSel.ParentsFiltered("a").First(); wrapper.Length() > 0 {
			image.AttachmentURL = client.ResolveURL(attrOrEmpty(wrapper, "href"))
		}

		images = append(images, image)
	})

	return images
}

func parseThreadPosts(client *auth.Client, body []byte) (string, []ThreadPost, string, error) {
	doc, err := newDocument(body)
	if err != nil {
		return "", nil, "", err
	}

	title := extractTitle(doc)
	next := nextPageURL(client, doc)
	posts := make([]ThreadPost, 0)

	doc.Find("article.message--post").Each(func(_ int, postSel *goquery.Selection) {
		post := ThreadPost{
			Author: cleanText(postSel.Find(".message-name .username").First().Text()),
		}

		if content := strings.TrimSpace(postSel.Find(".message-body .bbWrapper").First().Text()); content != "" {
			post.Content = content
		} else {
			post.Content = strings.TrimSpace(postSel.Find(".message-body").First().Text())
		}

		post.PostedAt = cleanText(postSel.Find(".message-attribution-main time").First().AttrOr("title", postSel.Find(".message-attribution-main time").First().Text()))
		postNumberLink := postSel.Find(".message-attribution-opposite a").Last()
		post.PostNumber = cleanText(postNumberLink.Text())
		post.PostURL = client.ResolveURL(attrOrEmpty(postNumberLink, "href"))
		post.Images = extractPostImages(client, postSel)

		if post.Author != "" {
			posts = append(posts, post)
		}
	})

	return title, posts, next, nil
}

func parseSearchResults(client *auth.Client, body []byte, query string, page int) (SearchResult, error) {
	doc, err := newDocument(body)
	if err != nil {
		return SearchResult{}, err
	}

	result := SearchResult{
		Query:       query,
		Page:        page,
		Results:     make([]SearchResultItem, 0),
		NextPageURL: nextPageURL(client, doc),
	}

	doc.Find(".contentRow").Each(func(_ int, row *goquery.Selection) {
		titleLink := row.Find(".contentRow-title a").First()
		title := cleanText(titleLink.Text())
		if title == "" {
			return
		}

		item := SearchResultItem{
			Title:   title,
			URL:     client.ResolveURL(attrOrEmpty(titleLink, "href")),
			Snippet: cleanText(row.Find(".contentRow-snippet").First().Text()),
		}
		result.Results = append(result.Results, item)
	})

	return result, nil
}
