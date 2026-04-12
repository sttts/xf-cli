package cmds

import (
	"fmt"

	"github.com/sttts/xf-mcp/scraper"
)

type ListThreadsCmd struct {
	ForumURL string `arg:"" required:"" help:"Forum URL or forum path."`
	Page     int    `default:"1" help:"Page number."`
}

func (cmd *ListThreadsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListThreads(client, session, cmd.ForumURL, cmd.Page)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
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
