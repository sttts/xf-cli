package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type SearchCmd struct {
	Query string `arg:"" required:"" help:"Search query."`
	Page  int    `default:"1" help:"Page number."`
}

func (cmd *SearchCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.Search(client, session, cmd.Query, cmd.Page)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
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
