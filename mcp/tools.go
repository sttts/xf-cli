package xfmcp

import (
	"context"

	mcpapi "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sttts/xf-cli/auth"
	"github.com/sttts/xf-cli/scraper"
)

type EmptyArgs struct{}

type ListThreadsArgs struct {
	ForumURL string `json:"forum_url" jsonschema:"Forum URL or forum path"`
	Page     string `json:"page,omitempty" jsonschema:"Page cursor returned by a previous call"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Minimum number of results to collect; 0 means all pages"`
}

type ReadThreadArgs struct {
	ThreadURL string `json:"thread_url" jsonschema:"Thread URL or thread path"`
}

type PageArgs struct {
	Page  string `json:"page,omitempty" jsonschema:"Page cursor returned by a previous call"`
	Limit int    `json:"limit,omitempty" jsonschema:"Minimum number of results to collect; 0 means all pages"`
}

type SearchArgs struct {
	Query string `json:"query" jsonschema:"Forum search query"`
	Page  string `json:"page,omitempty" jsonschema:"Page cursor returned by a previous call"`
	Limit int    `json:"limit,omitempty" jsonschema:"Minimum number of results to collect; 0 means all pages"`
}

type ProfileArgs struct {
	ProfileURL string `json:"profile_url" jsonschema:"Profile URL or member path"`
	Page       string `json:"page,omitempty" jsonschema:"Page cursor returned by a previous call"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Minimum number of results to collect; 0 means all pages"`
}

type FollowLinkArgs struct {
	URL string `json:"url" jsonschema:"Forum, thread, post, attachment or image URL/path"`
}

func Tools(config Config) ([]server.ServerTool, error) {
	provider, err := NewSessionProvider(config)
	if err != nil {
		return nil, err
	}

	return []server.ServerTool{
		{
			Tool: readOnlyTool(
				"list_forums",
				"List forum categories and forums visible to the authenticated user.",
				mcpapi.WithInputSchema[EmptyArgs](),
				mcpapi.WithOutputSchema[scraper.ForumListResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args EmptyArgs) (scraper.ForumListResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.ForumListResult, error) {
					return scraper.ListForums(client, session)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"list_threads",
				"List threads for a forum using cursor-based paging. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[ListThreadsArgs](),
				mcpapi.WithOutputSchema[scraper.ThreadListResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args ListThreadsArgs) (scraper.ThreadListResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.ThreadListResult, error) {
					return scraper.ListThreads(client, session, args.ForumURL, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"list_new_posts",
				"List new posts visible to the authenticated user using cursor-based paging. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[PageArgs](),
				mcpapi.WithOutputSchema[scraper.ThreadListResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args PageArgs) (scraper.ThreadListResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.ThreadListResult, error) {
					return scraper.ListNewPosts(client, session, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"read_thread",
				"Read a full thread across all pages, including extracted image references.",
				mcpapi.WithInputSchema[ReadThreadArgs](),
				mcpapi.WithOutputSchema[scraper.ThreadReadResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args ReadThreadArgs) (scraper.ThreadReadResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.ThreadReadResult, error) {
					return scraper.ReadThread(client, session, args.ThreadURL)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"search_threads",
				"Search thread titles and return paged thread-match results. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[SearchArgs](),
				mcpapi.WithOutputSchema[scraper.SearchResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args SearchArgs) (scraper.SearchResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.SearchResult, error) {
					return scraper.SearchThreads(client, session, args.Query, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"search_posts",
				"Search post contents and return snippet-level matches with cursor-based paging. Returns only short snippets, not full post bodies. For anything relevant, call read_thread on the matched thread URL before relying on the result. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[SearchArgs](),
				mcpapi.WithOutputSchema[scraper.SearchResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args SearchArgs) (scraper.SearchResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.SearchResult, error) {
					return scraper.SearchPosts(client, session, args.Query, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"read_profile",
				"Read a public profile page for a forum user.",
				mcpapi.WithInputSchema[ProfileArgs](),
				mcpapi.WithOutputSchema[scraper.UserProfile](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args ProfileArgs) (scraper.UserProfile, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.UserProfile, error) {
					return scraper.ReadProfile(client, session, args.ProfileURL)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"list_user_posts",
				"List recent public posts for a specific forum user using cursor-based paging. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[ProfileArgs](),
				mcpapi.WithOutputSchema[scraper.UserPostsResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args ProfileArgs) (scraper.UserPostsResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.UserPostsResult, error) {
					return scraper.ListUserPosts(client, session, args.ProfileURL, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"list_user_threads",
				"List threads started by a specific forum user using cursor-based paging. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[ProfileArgs](),
				mcpapi.WithOutputSchema[scraper.UserThreadsResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args ProfileArgs) (scraper.UserThreadsResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.UserThreadsResult, error) {
					return scraper.ListUserThreads(client, session, args.ProfileURL, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"list_my_threads",
				"List threads started by the authenticated user using cursor-based paging. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[PageArgs](),
				mcpapi.WithOutputSchema[scraper.ThreadListResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args PageArgs) (scraper.ThreadListResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.ThreadListResult, error) {
					return scraper.ListMyThreads(client, session, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"list_threads_i_participated",
				"List threads where the authenticated user has posted using cursor-based paging. Use page to continue from next_page and limit to cap how many results are collected.",
				mcpapi.WithInputSchema[PageArgs](),
				mcpapi.WithOutputSchema[scraper.ThreadListResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args PageArgs) (scraper.ThreadListResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.ThreadListResult, error) {
					return scraper.ListThreadsIParticipated(client, session, args.Page, args.Limit)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"follow_link",
				"Resolve and normalize an internal forum link to its canonical target.",
				mcpapi.WithInputSchema[FollowLinkArgs](),
				mcpapi.WithOutputSchema[scraper.FollowLinkResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args FollowLinkArgs) (scraper.FollowLinkResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.FollowLinkResult, error) {
					return scraper.FollowLink(client, session, args.URL)
				})
			}),
		},
		{
			Tool: readOnlyTool(
				"get_image",
				"Resolve an image or attachment URL to thumbnail and full image URLs.",
				mcpapi.WithInputSchema[FollowLinkArgs](),
				mcpapi.WithOutputSchema[scraper.ImageResult](),
			),
			Handler: mcpapi.NewStructuredToolHandler(func(ctx context.Context, request mcpapi.CallToolRequest, args FollowLinkArgs) (scraper.ImageResult, error) {
				return withSession(provider, func(client *auth.Client, session auth.SessionInfo) (scraper.ImageResult, error) {
					return scraper.GetImage(client, session, args.URL)
				})
			}),
		},
	}, nil
}

func withSession[T any](provider *SessionProvider, fn func(client *auth.Client, session auth.SessionInfo) (T, error)) (T, error) {
	client, session, err := provider.Login()
	if err != nil {
		var zero T
		return zero, err
	}

	return fn(client, session)
}
