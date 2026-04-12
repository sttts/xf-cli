package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/sttts/xf-mcp/auth"
	"github.com/sttts/xf-mcp/scraper"
	"golang.org/x/term"
)

type CLI struct {
	BaseURL  string `name:"base-url" default:"https://www.rc-network.de" help:"Forum base URL."`
	Username string `short:"u" help:"Forum username or email."`
	Password string `short:"p" help:"Forum password. Omit for secure prompt or XF_PASSWORD."`
	Verbose  bool   `short:"v" help:"Verbose logging of HTTP request/response."`
	AsJSON   bool   `name:"json" help:"Output command result as JSON."`

	Login       LoginCmd       `cmd:"" name:"login" help:"Perform login and print session info."`
	ListForums  ListForumsCmd  `cmd:"" name:"list_forums" help:"List forum categories and forums."`
	ListThreads ListThreadsCmd `cmd:"" name:"list_threads" help:"List threads for a forum."`
	ReadThread  ReadThreadCmd  `cmd:"" name:"read_thread" help:"Read a full thread across all pages."`
	Search      SearchCmd      `cmd:"" name:"search" help:"Search the forum."`
}

type LoginCmd struct{}

type ListForumsCmd struct{}

type ListThreadsCmd struct {
	ForumURL string `arg:"" required:"" help:"Forum URL or forum path."`
	Page     int    `default:"1" help:"Page number."`
}

type ReadThreadCmd struct {
	ThreadURL string `arg:"" required:"" help:"Thread URL or thread path."`
}

type SearchCmd struct {
	Query string `arg:"" required:"" help:"Search query."`
	Page  int    `default:"1" help:"Page number."`
}

type App struct {
	baseURL  string
	username string
	password string
	verbose  bool
	asJSON   bool
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func readPassword() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	bytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func readLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	var value string
	fmt.Scanln(&value)
	return value
}

func printJSON(value any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func newApp(cli *CLI) *App {
	return &App{
		baseURL:  cli.BaseURL,
		username: cli.Username,
		password: cli.Password,
		verbose:  cli.Verbose,
		asJSON:   cli.AsJSON,
	}
}

func (a *App) ensureCredentials() error {
	if a.username == "" {
		a.username = strings.TrimSpace(os.Getenv("XF_USERNAME"))
	}
	if a.username == "" {
		a.username = readLine("Username / Email: ")
	}

	if a.password == "" {
		a.password = strings.TrimSpace(os.Getenv("XF_PASSWORD"))
	}
	if a.password == "" {
		pass, err := readPassword()
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		a.password = pass
	}

	return nil
}

func (a *App) login() (*auth.Client, auth.SessionInfo, error) {
	if err := a.ensureCredentials(); err != nil {
		return nil, auth.SessionInfo{}, err
	}

	client, err := auth.NewClient(a.baseURL, a.verbose)
	if err != nil {
		return nil, auth.SessionInfo{}, err
	}

	session, err := client.Login(a.username, a.password)
	if err != nil {
		return nil, auth.SessionInfo{}, err
	}

	return client, session, nil
}

func (a *App) printSession(session auth.SessionInfo) error {
	if a.asJSON {
		return printJSON(session)
	}

	fmt.Printf("Logged in as:  %s\n", session.Username)
	fmt.Printf("CSRF Token:    %s\n", session.XFToken)
	fmt.Println("Cookies:")
	for key, value := range session.Cookies {
		fmt.Printf("  %s = %s\n", key, value)
	}
	fmt.Println()
	fmt.Println("Example curl with session:")
	cookieParts := make([]string, 0, len(session.Cookies))
	for key, value := range session.Cookies {
		cookieParts = append(cookieParts, key+"="+value)
	}
	fmt.Printf("  curl -b \"%s\" %s/account/\n", strings.Join(cookieParts, "; "), session.BaseURL)
	return nil
}

func (cmd *LoginCmd) Run(app *App) error {
	_, session, err := app.login()
	if err != nil {
		return err
	}

	return app.printSession(session)
}

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

func (cmd *ListThreadsCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ListThreads(client, session, cmd.ForumURL, cmd.Page)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Forum: %s\n", result.ForumTitle)
	fmt.Printf("Threads on page %d: %d\n", result.Page, len(result.Threads))
	for _, thread := range result.Threads {
		fmt.Printf("\n- %s\n", thread.Title)
		fmt.Printf("  %s\n", thread.URL)
		if thread.Author != "" {
			fmt.Printf("  by %s\n", thread.Author)
		}
	}
	if result.NextPageURL != "" {
		fmt.Printf("\nNext page: %s\n", result.NextPageURL)
	}
	return nil
}

func (cmd *ReadThreadCmd) Run(app *App) error {
	client, session, err := app.login()
	if err != nil {
		return err
	}

	result, err := scraper.ReadThread(client, session, cmd.ThreadURL)
	if err != nil {
		return err
	}

	if app.asJSON {
		return printJSON(result)
	}

	fmt.Printf("Logged in as: %s\n", result.Username)
	fmt.Printf("Thread: %s\n", result.Title)
	fmt.Printf("Pages read: %d\n", result.PagesRead)
	fmt.Printf("Posts: %d\n", len(result.Posts))
	for _, post := range result.Posts {
		fmt.Printf("\n%s %s\n", post.PostNumber, post.Author)
		if post.PostedAt != "" {
			fmt.Printf("%s\n", post.PostedAt)
		}
		fmt.Printf("%s\n", truncate(post.Content, 300))
		if len(post.Images) > 0 {
			fmt.Printf("Images: %d\n", len(post.Images))
		}
	}
	return nil
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

func main() {
	cli := CLI{}
	parser := kong.Parse(&cli,
		kong.Name("xf-mcp"),
		kong.Description("Read-only XenForo frontend client and future MCP tool backend."),
	)

	app := newApp(&cli)
	if err := parser.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
