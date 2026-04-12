package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ListThreadsCmd struct {
	ForumURL string `arg:"" required:"" help:"Forum URL or forum path."`
	Page     string `help:"Page cursor returned by a previous call."`
	Limit    int    `default:"50" help:"Minimum number of threads to collect; 0 means all pages."`
}

func (cmd *ListThreadsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListThreads(client, session, cmd.ForumURL, cmd.Page, cmd.Limit)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Forum: %s\n", result.ForumTitle)
	fmt.Printf("Threads from page %d: %d\n", result.Page, len(result.Threads))
	for _, thread := range result.Threads {
		fmt.Printf("\n- %s\n", thread.Title)
		fmt.Printf("  %s\n", thread.URL)
		if thread.Author != "" {
			fmt.Printf("  by %s\n", thread.Author)
		}
	}
	if result.NextPage != "" {
		fmt.Printf("\nNext page cursor: %s\n", result.NextPage)
	}
	if result.NextPageURL != "" {
		fmt.Printf("Next page URL: %s\n", result.NextPageURL)
	}

	return nil
}
