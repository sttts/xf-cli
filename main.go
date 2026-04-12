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
	baseURL    = flag.String("base-url", "https://www.rc-network.de", "Forum base URL")
	username   = flag.String("u", "", "Forum username or email")
	password   = flag.String("p", "", "Forum password (omit for secure prompt)")
	verbose    = flag.Bool("v", false, "Verbose logging of HTTP request/response")
	asJSON     = flag.Bool("json", false, "Output session info as JSON")
	listForums = flag.Bool("list-forums", false, "List forum categories and subforums after login")
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
