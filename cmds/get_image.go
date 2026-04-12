package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type GetImageCmd struct {
	URL string `arg:"" required:"" help:"Attachment or image URL/path."`
}

func (cmd *GetImageCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.GetImage(client, session, cmd.URL)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Canonical: %s\n", result.CanonicalURL)
	if result.AttachmentURL != "" {
		fmt.Printf("Attachment: %s\n", result.AttachmentURL)
	}
	if result.ThumbnailURL != "" {
		fmt.Printf("Thumbnail: %s\n", result.ThumbnailURL)
	}
	if result.FullImageURL != "" {
		fmt.Printf("Full image: %s\n", result.FullImageURL)
	}
	if result.ContentType != "" {
		fmt.Printf("Content-Type: %s\n", result.ContentType)
	}

	return nil
}
