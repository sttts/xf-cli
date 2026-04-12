package scraper

import (
	"fmt"
	"mime"
	"path"
	"strings"

	"github.com/sttts/xf-cli/auth"
)

type LinkType string

const (
	LinkTypeForum      LinkType = "forum"
	LinkTypeThread     LinkType = "thread"
	LinkTypePost       LinkType = "post"
	LinkTypeAttachment LinkType = "attachment"
	LinkTypeImage      LinkType = "image"
	LinkTypeExternal   LinkType = "external"
	LinkTypeUnknown    LinkType = "unknown"
)

type FollowLinkResult struct {
	Username      string   `json:"username"`
	BaseURL       string   `json:"base_url"`
	InputURL      string   `json:"input_url"`
	Type          LinkType `json:"type"`
	CanonicalURL  string   `json:"canonical_url"`
	ResolvedURL   string   `json:"resolved_url,omitempty"`
	ThreadURL     string   `json:"thread_url,omitempty"`
	PostURL       string   `json:"post_url,omitempty"`
	ForumURL      string   `json:"forum_url,omitempty"`
	AttachmentURL string   `json:"attachment_url,omitempty"`
	ImageURL      string   `json:"image_url,omitempty"`
	ContentType   string   `json:"content_type,omitempty"`
}

func FollowLink(client *auth.Client, session auth.SessionInfo, ref string) (FollowLinkResult, error) {
	inputURL := client.ResolveURL(ref)
	result := FollowLinkResult{
		Username: session.Username,
		BaseURL:  session.BaseURL,
		InputURL: inputURL,
	}

	if !strings.HasPrefix(inputURL, client.BaseURL()) {
		result.Type = LinkTypeExternal
		result.CanonicalURL = inputURL
		result.ResolvedURL = inputURL
		return result, nil
	}

	resp, err := client.HeadNoRedirect(inputURL)
	if err != nil {
		return result, fmt.Errorf("following link: %w", err)
	}
	defer resp.Body.Close()

	result.ContentType = resp.Header.Get("Content-Type")
	result.ResolvedURL = inputURL
	if location := resp.Header.Get("Location"); location != "" {
		result.ResolvedURL = client.ResolveURL(location)
	}

	switch {
	case strings.Contains(inputURL, "/forums/"):
		result.Type = LinkTypeForum
		result.ForumURL = trimURLFragment(client.ResolveURL(stripPageSuffix(inputURL)))
		result.CanonicalURL = result.ForumURL
	case strings.Contains(inputURL, "/threads/") && strings.Contains(inputURL, "/post-"):
		result.Type = LinkTypePost
		result.PostURL = trimURLFragment(inputURL)
		result.ThreadURL = canonicalThreadURL(inputURL)
		result.CanonicalURL = result.PostURL
	case strings.Contains(inputURL, "/threads/"):
		if strings.Contains(inputURL, "/latest") {
			result.Type = LinkTypePost
			result.PostURL = trimURLFragment(result.ResolvedURL)
			result.ThreadURL = canonicalThreadURL(result.ResolvedURL)
			result.CanonicalURL = result.PostURL
			break
		}
		result.Type = LinkTypeThread
		result.ThreadURL = canonicalThreadURL(inputURL)
		result.CanonicalURL = result.ThreadURL
	case strings.Contains(inputURL, "/attachments/"):
		result.Type = LinkTypeAttachment
		result.AttachmentURL = trimURLFragment(inputURL)
		result.CanonicalURL = result.AttachmentURL
		if isImageContentType(result.ContentType) || hasImageExtension(inputURL) {
			result.ImageURL = result.ResolvedURL
		}
	case strings.Contains(inputURL, "/data/attachments/"):
		result.Type = LinkTypeImage
		result.ImageURL = trimURLFragment(inputURL)
		result.CanonicalURL = result.ImageURL
	case isImageContentType(result.ContentType) || hasImageExtension(inputURL):
		result.Type = LinkTypeImage
		result.ImageURL = trimURLFragment(inputURL)
		result.CanonicalURL = result.ImageURL
	default:
		result.Type = LinkTypeUnknown
		result.CanonicalURL = trimURLFragment(inputURL)
	}

	return result, nil
}

func trimURLFragment(value string) string {
	if idx := strings.IndexByte(value, '#'); idx >= 0 {
		return value[:idx]
	}
	return value
}

func stripPageSuffix(value string) string {
	for _, marker := range []string{"/page-", "?page="} {
		if idx := strings.Index(value, marker); idx >= 0 {
			return value[:idx]
		}
	}
	return value
}

func canonicalThreadURL(value string) string {
	base := trimURLFragment(value)
	if idx := strings.Index(base, "/post-"); idx >= 0 {
		base = base[:idx]
	}
	for _, suffix := range []string{"/latest", "/unread"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
		}
	}
	base = stripPageSuffix(base)
	return base
}

func isImageContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	return strings.HasPrefix(mediaType, "image/")
}

func hasImageExtension(value string) bool {
	ext := strings.ToLower(path.Ext(trimURLFragment(value)))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}
