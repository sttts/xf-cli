package auth

import (
	"fmt"
	"net/url"
)

func (c *Client) Login(username, password string) (SessionInfo, error) {
	loginPageURL := c.baseURL + "/login/"
	body, err := c.FetchPage(loginPageURL)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("fetching login page: %w", err)
	}

	csrfToken, err := ExtractCSRFToken(body)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("extracting CSRF token from login page: %w", err)
	}
	c.logf("CSRF token: %s", csrfToken)

	loginURL := c.baseURL + "/login/login"
	form := url.Values{
		"login":       {username},
		"password":    {password},
		"remember":    {"1"},
		"_xfRedirect": {c.baseURL + "/"},
		"_xfToken":    {csrfToken},
	}

	body, err = c.PostForm(loginURL, form, loginPageURL)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("login request: %w", err)
	}

	if loginError := DetectLoginError(body); loginError != "" {
		return SessionInfo{}, fmt.Errorf("login failed: %s", loginError)
	}

	body, err = c.FetchPage(c.baseURL + "/")
	if err != nil {
		return SessionInfo{}, fmt.Errorf("fetching main page: %w", err)
	}

	if !IsLoggedIn(body) {
		return SessionInfo{}, fmt.Errorf("login appears to have failed – no session established")
	}

	freshToken := ""
	if token, err := ExtractCSRFToken(body); err == nil {
		freshToken = token
	}

	return SessionInfo{
		Username: username,
		Cookies:  c.Cookies(),
		XFToken:  freshToken,
		BaseURL:  c.baseURL,
	}, nil
}
