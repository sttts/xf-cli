package scraper

import (
	"fmt"

	"github.com/sttts/xf-cli/auth"
)

type ImageResult struct {
	Username      string `json:"username"`
	BaseURL       string `json:"base_url"`
	InputURL      string `json:"input_url"`
	CanonicalURL  string `json:"canonical_url"`
	AttachmentURL string `json:"attachment_url,omitempty"`
	ThumbnailURL  string `json:"thumbnail_url,omitempty"`
	FullImageURL  string `json:"full_image_url,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
}

func GetImage(client *auth.Client, session auth.SessionInfo, ref string) (ImageResult, error) {
	link, err := FollowLink(client, session, ref)
	if err != nil {
		return ImageResult{}, err
	}

	result := ImageResult{
		Username:     session.Username,
		BaseURL:      session.BaseURL,
		InputURL:     link.InputURL,
		CanonicalURL: link.CanonicalURL,
		ContentType:  link.ContentType,
	}

	switch link.Type {
	case LinkTypeAttachment:
		result.AttachmentURL = link.AttachmentURL
		if link.ImageURL != "" {
			result.FullImageURL = link.ImageURL
		} else {
			result.FullImageURL = link.AttachmentURL
		}
	case LinkTypeImage:
		result.ThumbnailURL = link.ImageURL
		result.FullImageURL = link.ImageURL
	default:
		return ImageResult{}, fmt.Errorf("link does not resolve to an image: %s", link.Type)
	}

	if result.CanonicalURL == "" {
		if result.AttachmentURL != "" {
			result.CanonicalURL = result.AttachmentURL
		} else {
			result.CanonicalURL = result.FullImageURL
		}
	}

	if result.ThumbnailURL == "" && result.FullImageURL != "" {
		result.ThumbnailURL = result.FullImageURL
	}

	return result, nil
}
