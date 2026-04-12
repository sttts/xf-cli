package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/sttts/xf-mcp/auth"
	"github.com/sttts/xf-mcp/scraper"
	"golang.org/x/term"
)

var (
	baseURL     = flag.String("base-url", "https://www.rc-network.de", "Forum base URL")
	username    = flag.String("u", "", "Forum username or email")
	password    = flag.String("p", "", "Forum password (omit for secure prompt)")
	verbose     = flag.Bool("v", false, "Verbose logging of HTTP request/response")
	asJSON      = flag.Bool("json", false, "Output session info as JSON")
	listForums  = flag.Bool("list-forums", false, "List forum categories and subforums after login")
	listThreads = flag.String("list-threads", "", "List threads for the given forum URL or path after login")
	readThread  = flag.String("read-thread", "", "Read all posts from the given thread URL or path after login")
	searchQuery = flag.String("search", "", "Search the forum for the given keywords after login")
	forumPage   = flag.Int("page", 1, "Page number for list-threads or search")
)

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

func run() error {
	client, err := auth.NewClient(*baseURL, *verbose)
	if err != nil {
		return err
	}

	session, err := client.Login(*username, *password)
	if err != nil {
		return err
	}

	page := *forumPage
	if page < 1 {
		page = 1
	}

	if *listForums {
		result, err := scraper.ListForums(client, session)
		if err != nil {
			return err
		}

		if *asJSON {
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

	if *listThreads != "" {
		result, err := scraper.ListThreads(client, session, *listThreads, page)
		if err != nil {
			return err
		}

		if *asJSON {
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

	if *readThread != "" {
		result, err := scraper.ReadThread(client, session, *readThread)
		if err != nil {
			return err
		}

		if *asJSON {
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

	if *searchQuery != "" {
		result, err := scraper.Search(client, session, *searchQuery, page)
		if err != nil {
			return err
		}

		if *asJSON {
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

	if *asJSON {
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

func main() {
	flag.Parse()

	user := *username
	if user == "" {
		user = strings.TrimSpace(os.Getenv("XF_USERNAME"))
	}
	if user == "" {
		user = readLine("Username / Email: ")
	}
	*username = user

	pass := *password
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv("XF_PASSWORD"))
	}
	if pass == "" {
		var err error
		pass, err = readPassword()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
	}
	*password = pass

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
