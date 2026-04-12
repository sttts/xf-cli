package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/auth"
	"github.com/sttts/xf-cli/scraper"
)

type SearchThreadsCmd struct {
	Query string `arg:"" required:"" help:"Search query."`
	Page  int    `default:"1" help:"Page number."`
}

func (cmd *SearchThreadsCmd) Run(app *App) error {
	return runSearch(app, cmd.Query, cmd.Page, scraper.SearchThreads)
}

type SearchPostsCmd struct {
	Query string `arg:"" required:"" help:"Search query."`
	Page  int    `default:"1" help:"Page number."`
}

func (cmd *SearchPostsCmd) Run(app *App) error {
	return runSearch(app, cmd.Query, cmd.Page, scraper.SearchPosts)
}

func runSearch(app *App, query string, page int, searchFunc func(client *auth.Client, session auth.SessionInfo, query string, page int) (scraper.SearchResult, error)) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := searchFunc(client, session, query, page)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Search type: %s\n", result.SearchType)
	fmt.Printf("Search query: %s\n", result.Query)
	fmt.Printf("Results: %d\n", len(result.Results))
	for _, item := range result.Results {
		fmt.Printf("\n- %s\n", item.Title)
		fmt.Printf("  %s\n", item.URL)
		if item.Snippet != "" {
			fmt.Printf("  %s\n", truncate(item.Snippet, 180))
		}
	}
	if result.NextPageURL != "" {
		fmt.Printf("\nNext page: %s\n", result.NextPageURL)
	}

	return nil
}
