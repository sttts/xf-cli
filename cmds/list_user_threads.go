package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ListUserThreadsCmd struct {
	UserURL string `arg:"" required:"" help:"User profile URL or path."`
	Page    string `help:"Page cursor returned by a previous call."`
	Limit   int    `default:"100" help:"Minimum number of threads to collect; 0 means all pages."`
}

func (cmd *ListUserThreadsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListUserThreads(client, session, cmd.UserURL, cmd.Page, cmd.Limit)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("User threads from page %d: %d\n", result.Page, len(result.Threads))
	for _, thread := range result.Threads {
		fmt.Printf("\n- %s\n", thread.Title)
		fmt.Printf("  %s\n", thread.URL)
	}
	if result.NextPage != "" {
		fmt.Printf("\nNext page cursor: %s\n", result.NextPage)
	}
	if result.NextPageURL != "" {
		fmt.Printf("Next page URL: %s\n", result.NextPageURL)
	}

	return nil
}
