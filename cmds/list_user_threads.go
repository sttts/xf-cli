package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ListUserThreadsCmd struct {
	UserURL string `arg:"" required:"" help:"User profile URL or path."`
	Page    int    `default:"1" help:"Page number."`
}

func (cmd *ListUserThreadsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListUserThreads(client, session, cmd.UserURL, cmd.Page)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("User threads page %d: %d\n", result.Page, len(result.Threads))
	for _, thread := range result.Threads {
		fmt.Printf("\n- %s\n", thread.Title)
		fmt.Printf("  %s\n", thread.URL)
	}
	if result.NextPageURL != "" {
		fmt.Printf("\nNext page: %s\n", result.NextPageURL)
	}

	return nil
}
