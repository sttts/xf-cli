package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ListNewPostsCmd struct {
	Page  string `help:"Page cursor returned by a previous call."`
	Limit int    `default:"100" help:"Minimum number of threads to collect; 0 means all pages."`
}

func (cmd *ListNewPostsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListNewPosts(client, session, cmd.Page, cmd.Limit)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("%s\n", result.ForumTitle)
	fmt.Printf("Threads from page %d: %d\n", result.Page, len(result.Threads))
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
