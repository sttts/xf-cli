package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/sttts/xf-mcp/auth"
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

type App struct {
	baseURL  string
	username string
	password string
	verbose  bool
	asJSON   bool
}

func Execute() error {
	cli := CLI{}
	parser := kong.Parse(&cli,
		kong.Name("xf-mcp"),
		kong.Description("Read-only XenForo frontend client and future MCP tool backend."),
	)

	app := &App{
		baseURL:  cli.BaseURL,
		username: cli.Username,
		password: cli.Password,
		verbose:  cli.Verbose,
		asJSON:   cli.AsJSON,
	}

	return parser.Run(app)
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
