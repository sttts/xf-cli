package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type FollowLinkCmd struct {
	URL string `arg:"" required:"" help:"Forum, thread, post, attachment or image URL/path."`
}

func (cmd *FollowLinkCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.FollowLink(client, session, cmd.URL)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Type: %s\n", result.Type)
	fmt.Printf("Canonical: %s\n", result.CanonicalURL)
	if result.ResolvedURL != "" && result.ResolvedURL != result.CanonicalURL {
		fmt.Printf("Resolved: %s\n", result.ResolvedURL)
	}
	if result.ThreadURL != "" {
		fmt.Printf("Thread: %s\n", result.ThreadURL)
	}
	if result.PostURL != "" {
		fmt.Printf("Post: %s\n", result.PostURL)
	}
	if result.ForumURL != "" {
		fmt.Printf("Forum: %s\n", result.ForumURL)
	}
	if result.AttachmentURL != "" {
		fmt.Printf("Attachment: %s\n", result.AttachmentURL)
	}
	if result.ImageURL != "" {
		fmt.Printf("Image: %s\n", result.ImageURL)
	}
	if result.ContentType != "" {
		fmt.Printf("Content-Type: %s\n", result.ContentType)
	}

	return nil
}
