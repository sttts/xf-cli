package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/term"
)

var (
	baseURL     = flag.String("base-url", "https://www.rc-network.de", "Forum base URL")
	username    = flag.String("u", "", "Forum username or email")
	password    = flag.String("p", "", "Forum password (omit for secure prompt)")
	verbose     = flag.Bool("v", false, "Verbose logging of HTTP request/response")
	asJSON      = flag.Bool("json", false, "Output session info as JSON")
	listForums  = flag.Bool("list-forums", false, "List forum categories and subforums after login")
	listThreads = flag.String("list-threads", "", "List threads for the given forum URL or path after login")
	readThread  = flag.String("read-thread", "", "Read all posts from the given thread URL or path after login")
	searchQuery = flag.String("search", "", "Search the forum for the given keywords after login")
	forumPage   = flag.Int("page", 1, "Page number for list-threads or search")
)

func logf(format string, a ...any) {
	if *verbose {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", a...)
	}
}

func logRequest(method, requestURL string, form url.Values) {
	logf("%s %s", method, requestURL)
	if form != nil {
		safe := make(url.Values)
		for k, v := range form {
			if k == "password" {
				safe[k] = []string{"***"}
			} else {
				safe[k] = v
			}
		}
		logf("Form: %s", safe.Encode())
	}
}

func logResponse(resp *http.Response, body []byte) {
	logf("Status: %s", resp.Status)
	for k, v := range resp.Header {
		logf("  %s: %s", k, strings.Join(v, ", "))
	}
	logf("Body (%d bytes): %s", len(body), truncate(string(body), 2000))
	logf("Cookies:")
	for _, c := range resp.Cookies() {
		logf("  %s=%s (path=%s)", c.Name, truncate(c.Value, 30), c.Path)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func parseDocument(body []byte) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	return doc, nil
}

func extractCSRFToken(body []byte) (string, error) {
	doc, err := parseDocument(body)
	if err != nil {
		return "", err
	}

	if token, ok := doc.Find(`input[name="_xfToken"]`).First().Attr("value"); ok && token != "" {
		return token, nil
	}

	if token, ok := doc.Find("html").First().Attr("data-csrf"); ok && token != "" {
		return token, nil
	}

	return "", fmt.Errorf("could not find CSRF token in HTML")
}

func detectLoginError(body []byte) string {
	doc, err := parseDocument(body)
	if err == nil {
		if msg := strings.TrimSpace(doc.Find(".blockMessage--error").First().Text()); msg != "" {
			return msg
		}

		if msg := strings.TrimSpace(doc.Find(".formRow--error").First().Text()); msg != "" {
			return msg
		}
	}

	bodyText := string(body)
	switch {
	case strings.Contains(bodyText, "Der angeforderte Benutzer") && strings.Contains(bodyText, "wurde nicht gefunden"):
		return "Der angeforderte Benutzer wurde nicht gefunden."
	case strings.Contains(bodyText, "Incorrect password"), strings.Contains(bodyText, "Falsches Passwort"):
		return "Falsches Passwort."
	default:
		return ""
	}
}

func isLoggedIn(body []byte) bool {
	doc, err := parseDocument(body)
	if err == nil {
		if state, ok := doc.Find("html").First().Attr("data-logged-in"); ok && state == "true" {
			return true
		}

		if doc.Find(`a[href*="/logout"], a[href*="/log-out"]`).Length() > 0 {
			return true
		}
	}

	bodyText := string(body)
	return strings.Contains(bodyText, `data-logged-in="true"`) ||
		strings.Contains(bodyText, "Abmelden") ||
		strings.Contains(bodyText, "log-out")
}

func readPassword() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func readLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	var s string
	fmt.Scanln(&s)
	return s
}

type SessionInfo struct {
	Username string            `json:"username"`
	Cookies  map[string]string `json:"cookies"`
	XFToken  string            `json:"xf_token"`
	BaseURL  string            `json:"base_url"`
}

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
	Title       string `json:"title"`
	URL         string `json:"url"`
	Author      string `json:"author,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	Replies     string `json:"replies,omitempty"`
	Views       string `json:"views,omitempty"`
	LastPostAt  string `json:"last_post_at,omitempty"`
	LastPoster  string `json:"last_poster,omitempty"`
}

type ThreadListResult struct {
	Username    string          `json:"username"`
	BaseURL     string          `json:"base_url"`
	ForumTitle  string          `json:"forum_title"`
	ForumURL    string          `json:"forum_url"`
	Page        int             `json:"page"`
	Threads     []ThreadSummary `json:"threads"`
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
	Query       string             `json:"query"`
	Page        int                `json:"page"`
	Results     []SearchResultItem `json:"results"`
	NextPageURL string             `json:"next_page_url,omitempty"`
}

func newRequest(method, requestURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")
	return req, nil
}

func fetchPage(client *http.Client, requestURL string) ([]byte, error) {
	logRequest("GET", requestURL, nil)

	req, err := newRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GET request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	logResponse(resp, body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", requestURL, resp.Status)
	}

	return body, nil
}

func postForm(client *http.Client, requestURL string, form url.Values, referer string) ([]byte, error) {
	logRequest("POST", requestURL, form)

	req, err := newRequest(http.MethodPost, requestURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating POST request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", strings.TrimRight(*baseURL, "/"))
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	logResponse(resp, body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", requestURL, resp.Status)
	}

	return body, nil
}

func absoluteURL(base, href string) string {
	if href == "" {
		return ""
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return href
	}

	refURL, err := url.Parse(href)
	if err != nil {
		return href
	}

	return baseURL.ResolveReference(refURL).String()
}

func forumURL(base, forumRef string, page int) string {
	u := absoluteURL(base, forumRef)
	if page <= 1 {
		return u
	}

	return strings.TrimRight(u, "/") + fmt.Sprintf("/page-%d", page)
}

func threadURL(base, threadRef string, page int) string {
	u := absoluteURL(base, threadRef)
	if page <= 1 {
		return u
	}

	return strings.TrimRight(u, "/") + fmt.Sprintf("/page-%d", page)
}

func cleanText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func extractTitle(doc *goquery.Document) string {
	return cleanText(doc.Find(".p-title-value").First().Text())
}

func nextPageURL(base string, doc *goquery.Document) string {
	if href, ok := doc.Find(".pageNav-jump--next").First().Attr("href"); ok && href != "" {
		return absoluteURL(base, href)
	}

	if href, ok := doc.Find(".pageNavSimple-el--next").First().Attr("href"); ok && href != "" {
		return absoluteURL(base, href)
	}

	return ""
}

func parseForumList(base string, body []byte) ([]ForumCategory, error) {
	doc, err := parseDocument(body)
	if err != nil {
		return nil, err
	}

	categories := make([]ForumCategory, 0)
	doc.Find(".node.node--category").Each(func(i int, categorySel *goquery.Selection) {
		category := ForumCategory{
			Forums: make([]ForumNode, 0),
		}

		if titleLink := categorySel.Find(".node-title a").First(); titleLink.Length() > 0 {
			category.Title = strings.TrimSpace(titleLink.Text())
			if href, ok := titleLink.Attr("href"); ok {
				category.URL = absoluteURL(base, href)
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
				forum.URL = absoluteURL(base, href)
			}
			category.Forums = append(category.Forums, forum)
		})

		if category.Title != "" && len(category.Forums) > 0 {
			categories = append(categories, category)
		}
	})

	return categories, nil
}

func parseThreadList(base string, body []byte, page int) (ThreadListResult, error) {
	doc, err := parseDocument(body)
	if err != nil {
		return ThreadListResult{}, err
	}

	result := ThreadListResult{
		ForumTitle:  extractTitle(doc),
		ForumURL:    "",
		Page:        page,
		Threads:     make([]ThreadSummary, 0),
		NextPageURL: nextPageURL(base, doc),
	}

	doc.Find(".structItem--thread").Each(func(_ int, item *goquery.Selection) {
		titleLink := item.Find(".structItem-title a").First()
		title := cleanText(titleLink.Text())
		if title == "" {
			return
		}

		thread := ThreadSummary{
			Title: title,
			URL:   absoluteURL(base, attrOrEmpty(titleLink, "href")),
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

func attrOrEmpty(sel *goquery.Selection, attr string) string {
	value, _ := sel.Attr(attr)
	return value
}

func extractPostImages(base string, post *goquery.Selection) []PostImage {
	images := make([]PostImage, 0)
	seen := make(map[string]struct{})

	post.Find(".message-userContent img").Each(func(_ int, imgSel *goquery.Selection) {
		className := imgSel.AttrOr("class", "")
		if strings.Contains(className, "smilie") || strings.Contains(className, "avatar") || strings.Contains(className, "reaction-image") {
			return
		}

		src := absoluteURL(base, attrOrEmpty(imgSel, "src"))
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
			image.AttachmentURL = absoluteURL(base, attrOrEmpty(link, "href"))
		} else if wrapper := imgSel.ParentsFiltered("a").First(); wrapper.Length() > 0 {
			image.AttachmentURL = absoluteURL(base, attrOrEmpty(wrapper, "href"))
		}

		images = append(images, image)
	})

	return images
}

func parseThreadPosts(base string, body []byte) (string, []ThreadPost, string, error) {
	doc, err := parseDocument(body)
	if err != nil {
		return "", nil, "", err
	}

	title := extractTitle(doc)
	next := nextPageURL(base, doc)
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
		post.PostURL = absoluteURL(base, attrOrEmpty(postNumberLink, "href"))
		post.Images = extractPostImages(base, postSel)

		if post.Author != "" {
			posts = append(posts, post)
		}
	})

	return title, posts, next, nil
}

func parseSearchResults(base string, body []byte, query string, page int) (SearchResult, error) {
	doc, err := parseDocument(body)
	if err != nil {
		return SearchResult{}, err
	}

	result := SearchResult{
		Query:       query,
		Page:        page,
		Results:     make([]SearchResultItem, 0),
		NextPageURL: nextPageURL(base, doc),
	}

	doc.Find(".contentRow").Each(func(_ int, row *goquery.Selection) {
		titleLink := row.Find(".contentRow-title a").First()
		title := cleanText(titleLink.Text())
		if title == "" {
			return
		}

		item := SearchResultItem{
			Title:   title,
			URL:     absoluteURL(base, attrOrEmpty(titleLink, "href")),
			Snippet: cleanText(row.Find(".contentRow-snippet").First().Text()),
		}
		result.Results = append(result.Results, item)
	})

	return result, nil
}

func run() error {
	base := strings.TrimRight(*baseURL, "/")

	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("creating cookie jar: %w", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			logf("Redirect: %s", req.URL)
			return nil
		},
	}

	client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	// Step 1: GET login page to obtain CSRF token and initial cookies
	loginPageURL := base + "/login/"
	body, err := fetchPage(client, loginPageURL)
	if err != nil {
		return fmt.Errorf("fetching login page: %w", err)
	}

	// Extract CSRF token
	csrfToken, err := extractCSRFToken(body)
	if err != nil {
		return fmt.Errorf("extracting CSRF token from login page: %w", err)
	}
	logf("CSRF token: %s", csrfToken)

	// Step 2: POST login credentials
	loginURL := base + "/login/login"
	form := url.Values{
		"login":         {*username},
		"password":      {*password},
		"remember":      {"1"},
		"_xfRedirect":   {base + "/"},
		"_xfToken":      {csrfToken},
	}
	logRequest("POST", loginURL, form)

	req, err := newRequest(http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", base)
	req.Header.Set("Referer", loginPageURL)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	logResponse(resp, body)

	// Check for login errors in response body
	if loginError := detectLoginError(body); loginError != "" {
		return fmt.Errorf("login failed: %s", loginError)
	}

	// Step 3: Verify login by fetching the main page and extracting session info
	body, err = fetchPage(client, base+"/")
	if err != nil {
		return fmt.Errorf("fetching main page: %w", err)
	}

	// Check if we're logged in (look for logout link or username)
	if !isLoggedIn(body) {
		return fmt.Errorf("login appears to have failed – no session established")
	}

	// Extract fresh CSRF token from authenticated page
	freshToken := ""
	if token, err := extractCSRFToken(body); err == nil {
		freshToken = token
	}

	// Collect session cookies
	u, _ := url.Parse(base)
	cookies := make(map[string]string)
	for _, c := range jar.Cookies(u) {
		cookies[c.Name] = c.Value
	}

	info := SessionInfo{
		Username: *username,
		Cookies:  cookies,
		XFToken:  freshToken,
		BaseURL:  base,
	}

	page := *forumPage
	if page < 1 {
		page = 1
	}

	if *listForums {
		forumsBody, err := fetchPage(client, base+"/forums/")
		if err != nil {
			return fmt.Errorf("fetching forum list: %w", err)
		}

		categories, err := parseForumList(base, forumsBody)
		if err != nil {
			return fmt.Errorf("parsing forum list: %w", err)
		}

		result := ForumListResult{
			Username:   info.Username,
			BaseURL:    info.BaseURL,
			Categories: categories,
		}

		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Printf("Logged in as: %s\n", result.Username)
		fmt.Printf("Forum categories: %d\n", len(result.Categories))
		for _, category := range result.Categories {
			fmt.Printf("\n%s\n", category.Title)
			for _, forum := range category.Forums {
				fmt.Printf("  - %s\n", forum.Title)
			}
		}

		return nil
	}

	if *listThreads != "" {
		forumRef := forumURL(base, *listThreads, page)
		threadsBody, err := fetchPage(client, forumRef)
		if err != nil {
			return fmt.Errorf("fetching thread list: %w", err)
		}

		result, err := parseThreadList(base, threadsBody, page)
		if err != nil {
			return fmt.Errorf("parsing thread list: %w", err)
		}
		result.Username = info.Username
		result.BaseURL = info.BaseURL
		result.ForumURL = forumRef

		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Printf("Logged in as: %s\n", result.Username)
		fmt.Printf("Forum: %s\n", result.ForumTitle)
		fmt.Printf("Threads on page %d: %d\n", result.Page, len(result.Threads))
		for _, thread := range result.Threads {
			fmt.Printf("\n- %s\n", thread.Title)
			fmt.Printf("  %s\n", thread.URL)
			if thread.Author != "" {
				fmt.Printf("  by %s\n", thread.Author)
			}
		}
		if result.NextPageURL != "" {
			fmt.Printf("\nNext page: %s\n", result.NextPageURL)
		}

		return nil
	}

	if *readThread != "" {
		currentURL := threadURL(base, *readThread, 1)
		seen := make(map[string]struct{})
		posts := make([]ThreadPost, 0)
		title := ""
		pagesRead := 0

		for currentURL != "" {
			if _, ok := seen[currentURL]; ok {
				break
			}
			seen[currentURL] = struct{}{}

			threadBody, err := fetchPage(client, currentURL)
			if err != nil {
				return fmt.Errorf("fetching thread page %s: %w", currentURL, err)
			}

			pageTitle, pagePosts, nextURL, err := parseThreadPosts(base, threadBody)
			if err != nil {
				return fmt.Errorf("parsing thread page %s: %w", currentURL, err)
			}

			if title == "" {
				title = pageTitle
			}
			posts = append(posts, pagePosts...)
			pagesRead++
			currentURL = nextURL
		}

		result := ThreadReadResult{
			Username:  info.Username,
			BaseURL:   info.BaseURL,
			ThreadURL: absoluteURL(base, *readThread),
			Title:     title,
			Posts:     posts,
			PagesRead: pagesRead,
		}

		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Printf("Logged in as: %s\n", result.Username)
		fmt.Printf("Thread: %s\n", result.Title)
		fmt.Printf("Pages read: %d\n", result.PagesRead)
		fmt.Printf("Posts: %d\n", len(result.Posts))
		for _, post := range result.Posts {
			fmt.Printf("\n%s %s\n", post.PostNumber, post.Author)
			if post.PostedAt != "" {
				fmt.Printf("%s\n", post.PostedAt)
			}
			fmt.Printf("%s\n", truncate(post.Content, 300))
			if len(post.Images) > 0 {
				fmt.Printf("Images: %d\n", len(post.Images))
			}
		}

		return nil
	}

	if *searchQuery != "" {
		searchPageURL := base + "/search/"
		searchPageBody, err := fetchPage(client, searchPageURL)
		if err != nil {
			return fmt.Errorf("fetching search page: %w", err)
		}

		token, err := extractCSRFToken(searchPageBody)
		if err != nil {
			return fmt.Errorf("extracting search token: %w", err)
		}

		form := url.Values{
			"keywords": {*searchQuery},
			"_xfToken": {token},
		}

		searchBody, err := postForm(client, base+"/search/search", form, searchPageURL)
		if err != nil {
			return fmt.Errorf("search request failed: %w", err)
		}

		result, err := parseSearchResults(base, searchBody, *searchQuery, 1)
		if err != nil {
			return fmt.Errorf("parsing search results: %w", err)
		}

		for currentPage := 1; currentPage < page && result.NextPageURL != ""; currentPage++ {
			searchBody, err = fetchPage(client, result.NextPageURL)
			if err != nil {
				return fmt.Errorf("fetching search page %d: %w", currentPage+1, err)
			}

			result, err = parseSearchResults(base, searchBody, *searchQuery, currentPage+1)
			if err != nil {
				return fmt.Errorf("parsing search page %d: %w", currentPage+1, err)
			}
		}

		result.Username = info.Username
		result.BaseURL = info.BaseURL

		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Printf("Logged in as: %s\n", result.Username)
		fmt.Printf("Search query: %s\n", result.Query)
		fmt.Printf("Results: %d\n", len(result.Results))
		for _, item := range result.Results {
			fmt.Printf("\n- %s\n", item.Title)
			fmt.Printf("  %s\n", item.URL)
			if item.Snippet != "" {
				fmt.Printf("  %s\n", truncate(item.Snippet, 180))
			}
		}
		if result.NextPageURL != "" {
			fmt.Printf("\nNext page: %s\n", result.NextPageURL)
		}

		return nil
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}

	fmt.Printf("Logged in as:  %s\n", info.Username)
	fmt.Printf("CSRF Token:    %s\n", info.XFToken)
	fmt.Println("Cookies:")
	for k, v := range info.Cookies {
		fmt.Printf("  %s = %s\n", k, v)
	}
	fmt.Println()
	fmt.Println("Example curl with session:")
	cookieParts := make([]string, 0, len(info.Cookies))
	for k, v := range info.Cookies {
		cookieParts = append(cookieParts, k+"="+v)
	}
	fmt.Printf("  curl -b \"%s\" %s/account/\n", strings.Join(cookieParts, "; "), base)

	return nil
}

func main() {
	flag.Parse()

	user := *username
	if user == "" {
		user = strings.TrimSpace(os.Getenv("XF_USERNAME"))
	}
	if user == "" {
		user = readLine("Username / Email: ")
	}
	*username = user

	pass := *password
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv("XF_PASSWORD"))
	}
	if pass == "" {
		var err error
		pass, err = readPassword()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
	}
	*password = pass

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
