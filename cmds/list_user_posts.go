package cmds

import (
	"fmt"

	"github.com/sttts/xf-cli/scraper"
)

type ListUserPostsCmd struct {
	UserURL string `arg:"" required:"" help:"User profile URL or path."`
	Page    string `help:"Page cursor returned by a previous call."`
	Limit   int    `default:"100" help:"Minimum number of posts to collect; 0 means all pages."`
}

func (cmd *ListUserPostsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListUserPosts(client, session, cmd.UserURL, cmd.Page, cmd.Limit)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("User posts from page %d: %d\n", result.Page, len(result.Posts))
	for _, post := range result.Posts {
		fmt.Printf("\n- %s\n", post.Title)
		fmt.Printf("  %s\n", post.PostURL)
		if post.ThreadURL != "" {
			fmt.Printf("  Thread: %s\n", post.ThreadURL)
		}
		if post.ForumTitle != "" {
			fmt.Printf("  Forum: %s\n", post.ForumTitle)
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
