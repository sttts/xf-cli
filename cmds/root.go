package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/sttts/xf-cli/auth"
	xfmcp "github.com/sttts/xf-cli/mcp"
	"golang.org/x/term"
)

type CLI struct {
	BaseURL  string `name:"base-url" default:"https://www.rc-network.de" help:"Forum base URL."`
	Username string `short:"u" help:"Forum username or email."`
	Password string `short:"p" help:"Forum password. Omit for secure prompt or XF_PASSWORD."`
	Verbose  int    `short:"v" type:"counter" help:"Increase HTTP debug logging. -v=requests, -vv=headers/status, -vvv=bodies."`
	AsJSON   bool   `name:"json" help:"Output command result as JSON."`

	Login                    LoginCmd                    `cmd:"" name:"login" help:"Perform login and print session info."`
	ListForums               ListForumsCmd               `cmd:"" name:"list_forums" help:"List forum categories and forums."`
	ListThreads              ListThreadsCmd              `cmd:"" name:"list_threads" help:"List threads for a forum."`
	ReadThread               ReadThreadCmd               `cmd:"" name:"read_thread" help:"Read a full thread across all pages."`
	SearchThreads            SearchThreadsCmd            `cmd:"" name:"search_threads" help:"Search thread titles."`
	SearchPosts              SearchPostsCmd              `cmd:"" name:"search_posts" help:"Search post contents."`
	ReadProfile              ReadProfileCmd              `cmd:"" name:"read_profile" help:"Read a public user profile."`
	ListUserPosts            ListUserPostsCmd            `cmd:"" name:"list_user_posts" help:"List a user's public posts."`
	ListUserThreads          ListUserThreadsCmd          `cmd:"" name:"list_user_threads" help:"List threads started by a user."`
	ListMyThreads            ListMyThreadsCmd            `cmd:"" name:"list_my_threads" help:"List threads started by the authenticated user."`
	ListThreadsIParticipated ListThreadsIParticipatedCmd `cmd:"" name:"list_threads_i_participated" help:"List threads with posts by the authenticated user."`
	FollowLink               FollowLinkCmd               `cmd:"" name:"follow_link" help:"Resolve and normalize an internal forum link."`
	GetImage                 GetImageCmd                 `cmd:"" name:"get_image" help:"Resolve image and attachment URLs."`
	MCP                      MCPCommand                  `cmd:"" name:"mcp" help:"Run the MCP server over stdin/stdout."`
}

type App struct {
	baseURL     string
	username    string
	password    string
	verbose     int
	asJSON      bool
	sessionPath string
}

func Execute() error {
	cli := CLI{}
	parser := kong.Must(&cli,
		kong.Name("xf-cli"),
		kong.Description("Read-only XenForo frontend client and future MCP tool backend."),
	)

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(parser.Stderr, "Error: %v\n\n", err)
		if parseErr, ok := err.(*kong.ParseError); ok {
			_ = parseErr.Context.PrintUsage(false)
		}
		return err
	}

	app := &App{
		baseURL:  cli.BaseURL,
		username: cli.Username,
		password: cli.Password,
		verbose:  cli.Verbose,
		asJSON:   cli.AsJSON,
	}
	if sessionPath, err := auth.DefaultSessionPath(); err == nil {
		app.sessionPath = sessionPath
	}

	if err := ctx.Run(app); err != nil {
		if _, ok := err.(*kong.ParseError); ok {
			fmt.Fprintf(parser.Stderr, "Error: %v\n\n", err)
			_ = ctx.PrintUsage(false)
			return err
		}

		return err
	}

	return nil
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
	client, err := auth.NewClient(a.baseURL, a.verbose)
	if err != nil {
		return nil, auth.SessionInfo{}, err
	}

	if a.sessionPath != "" {
		if stored, err := auth.LoadSession(a.sessionPath); err == nil && stored.BaseURL == a.baseURL {
			session, err := client.VerifySession(stored)
			if err == nil {
				_ = auth.SaveSession(a.sessionPath, session)
				return client, session, nil
			}
		}
	}

	if err := a.ensureCredentials(); err != nil {
		return nil, auth.SessionInfo{}, err
	}

	session, err := client.Login(a.username, a.password)
	if err != nil {
		return nil, auth.SessionInfo{}, err
	}
	if a.sessionPath != "" {
		_ = auth.SaveSession(a.sessionPath, session)
	}

	return client, session, nil
}

func (a *App) mcpConfig() (xfmcp.Config, error) {
	username := a.username
	if username == "" {
		username = strings.TrimSpace(os.Getenv("XF_USERNAME"))
	}

	password := a.password
	if password == "" {
		password = strings.TrimSpace(os.Getenv("XF_PASSWORD"))
	}

	cfg := xfmcp.Config{
		BaseURL:  a.baseURL,
		Username: username,
		Password: password,
		Verbose:  a.verbose,
	}

	if err := cfg.Validate(); err != nil {
		return xfmcp.Config{}, err
	}

	return cfg, nil
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
