package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ListMyThreadsCmd struct {
	Page int `default:"1" help:"Page number."`
}

func (cmd *ListMyThreadsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListMyThreads(client, session, cmd.Page)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("%s\n", result.ForumTitle)
	fmt.Printf("Threads on page %d: %d\n", result.Page, len(result.Threads))
	for _, thread := range result.Threads {
		fmt.Printf("\n- %s\n", thread.Title)
		fmt.Printf("  %s\n", thread.URL)
	}
	if result.NextPageURL != "" {
		fmt.Printf("\nNext page: %s\n", result.NextPageURL)
	}

	return nil
}
