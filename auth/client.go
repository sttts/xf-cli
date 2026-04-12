package auth

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	verbose    bool
	logWriter  io.Writer
	mu         sync.Mutex
	nextReqAt  time.Time
	interval   time.Duration
}

type SessionInfo struct {
	Username string            `json:"username"`
	Cookies  map[string]string `json:"cookies"`
	XFToken  string            `json:"xf_token"`
	BaseURL  string            `json:"base_url"`
}

func NewClient(baseURL string, verbose bool) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	client := &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		verbose:   verbose,
		logWriter: os.Stderr,
		interval:  200 * time.Millisecond,
	}

	client.httpClient = &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			client.logf("Redirect: %s", req.URL)
			return nil
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	return client, nil
}

func (c *Client) waitTurn() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if wait := time.Until(c.nextReqAt); wait > 0 {
		c.logf("Rate limit sleep: %s", wait.Truncate(time.Millisecond))
		time.Sleep(wait)
		now = time.Now()
	}

	c.nextReqAt = now.Add(c.interval)
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) logf(format string, args ...any) {
	if c.verbose {
		fmt.Fprintf(c.logWriter, "[debug] "+format+"\n", args...)
	}
}

func (c *Client) logRequest(method, requestURL string, form url.Values) {
	c.logf("%s %s", method, requestURL)
	if form == nil {
		return
	}

	safe := make(url.Values)
	for key, values := range form {
		if key == "password" {
			safe[key] = []string{"***"}
			continue
		}
		safe[key] = values
	}
	c.logf("Form: %s", safe.Encode())
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func (c *Client) logResponse(resp *http.Response, body []byte) {
	c.logf("Status: %s", resp.Status)
	for key, values := range resp.Header {
		c.logf("  %s: %s", key, strings.Join(values, ", "))
	}
	c.logf("Body (%d bytes): %s", len(body), truncate(string(body), 2000))
	c.logf("Cookies:")
	for _, cookie := range resp.Cookies() {
		c.logf("  %s=%s (path=%s)", cookie.Name, truncate(cookie.Value, 30), cookie.Path)
	}
}

func newDocument(body []byte) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	return doc, nil
}

func ExtractCSRFToken(body []byte) (string, error) {
	doc, err := newDocument(body)
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

func DetectLoginError(body []byte) string {
	doc, err := newDocument(body)
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

func IsLoggedIn(body []byte) bool {
	doc, err := newDocument(body)
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

func (c *Client) newRequest(method, requestURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")
	return req, nil
}

func (c *Client) FetchPage(requestURL string) ([]byte, error) {
	c.logRequest("GET", requestURL, nil)
	c.waitTurn()

	req, err := c.newRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GET request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	c.logResponse(resp, body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", requestURL, resp.Status)
	}

	return body, nil
}

func (c *Client) PostForm(requestURL string, form url.Values, referer string) ([]byte, error) {
	c.logRequest("POST", requestURL, form)
	c.waitTurn()

	req, err := c.newRequest(http.MethodPost, requestURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating POST request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", c.baseURL)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	c.logResponse(resp, body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", requestURL, resp.Status)
	}

	return body, nil
}

func (c *Client) HeadNoRedirect(requestURL string) (*http.Response, error) {
	c.logRequest("HEAD", requestURL, nil)
	c.waitTurn()

	req, err := c.newRequest(http.MethodHead, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HEAD request: %w", err)
	}

	noRedirectClient := *c.httpClient
	noRedirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return nil, err
	}

	c.logf("Status: %s", resp.Status)
	for key, values := range resp.Header {
		c.logf("  %s: %s", key, strings.Join(values, ", "))
	}

	return resp, nil
}

func (c *Client) ResolveURL(href string) string {
	if href == "" {
		return ""
	}

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return href
	}

	ref, err := url.Parse(href)
	if err != nil {
		return href
	}

	return base.ResolveReference(ref).String()
}

func (c *Client) Cookies() map[string]string {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil
	}

	cookies := make(map[string]string)
	for _, cookie := range c.httpClient.Jar.Cookies(base) {
		cookies[cookie.Name] = cookie.Value
	}

	return cookies
}
