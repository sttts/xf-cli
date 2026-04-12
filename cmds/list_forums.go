package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ListForumsCmd struct{}

func (cmd *ListForumsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListForums(client, session)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Forum categories: %d\n", len(result.Categories))
	for _, category := range result.Categories {
		fmt.Printf("\n%s\n", category.Title)
		for _, forum := range category.Forums {
			fmt.Printf("  - %s\n", forum.Title)
		}
	}

	return nil
}
