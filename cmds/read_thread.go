package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ReadThreadCmd struct {
	ThreadURL string `arg:"" required:"" help:"Thread URL or thread path."`
}

func (cmd *ReadThreadCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ReadThread(client, session, cmd.ThreadURL)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
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
