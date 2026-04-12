package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ReadProfileCmd struct {
	UserURL string `arg:"" required:"" help:"User profile URL or path."`
}

func (cmd *ReadProfileCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ReadProfile(client, session, cmd.UserURL)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Profile: %s\n", result.DisplayName)
	if result.UserTitle != "" {
		fmt.Printf("Title: %s\n", result.UserTitle)
	}
	if result.JoinedAt != "" {
		fmt.Printf("Joined: %s\n", result.JoinedAt)
	}
	if result.LastActivity != "" {
		fmt.Printf("Last activity: %s\n", result.LastActivity)
	}
	if result.PostCount != "" {
		fmt.Printf("Posts: %s\n", result.PostCount)
	}
	if result.ReactionScore != "" {
		fmt.Printf("Reactions: %s\n", result.ReactionScore)
	}

	return nil
}
