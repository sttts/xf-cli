# XenForo MCP Server Design

## Goal

Build an MCP server in Go that logs into a XenForo 2.x forum as a normal user and exposes forum content to LLMs over stdio JSON-RPC.

Target forum for validation:
- `https://www.rc-network.de`

Key constraint:
- no XenForo REST API key
- no admin privileges
- authentication must work through the normal browser login flow

## Current PoC Status

The PoC in [`main.go`](./main.go) has already validated these assumptions against `rc-network.de`:

- browser-style login with username/password works
- a logged-in session yields valid XenForo cookies and a fresh `_xfToken`
- XenForo REST API exists, but still rejects normal session cookies without `XF-Api-Key`
- forum overview scraping works
- forum thread list scraping works
- full thread reading across all pages works
- image extraction from posts works
- `search_threads` works
- `search_posts` works

This means the project direction is confirmed:
- use frontend HTML, not XenForo REST API

## Non-Goals

For the first implementation phase:

- read-only server
- no admin-only functionality
- no API-key-based XenForo integration
- no private-message support
- no automatic loading of `.env`
- no hidden in-memory task tracking

## User-Facing MCP Tools

The first useful MCP slice should expose these tools.

All MCP transport for this project is:
- stdio only
- JSON-RPC 2.0 in MCP form
- no HTTP server mode

The CLI subcommands are expected to mirror the MCP tool names exactly.

## MCP Tool Contracts

The MCP server should expose the same logical operations that already exist in the CLI and scraper layer.

Shared conventions:
- all URLs returned by tools should be absolute URLs
- all tools are read-only
- pagination tools return one page at a time unless explicitly documented otherwise
- `read_thread` is intentionally multi-page and returns the whole thread

### Common Error Shape

Tool failures should map to a stable internal shape before being translated to MCP tool errors.

Minimum fields:
- `code`
- `message`
- optional `details`

Suggested stable codes:
- `auth_login_failed`
- `auth_session_expired`
- `csrf_missing`
- `http_error`
- `page_structure_changed`
- `not_found`
- `forbidden`
- `rate_limited`
- `unsupported_link_type`
- `not_an_image`

### Common Input Conventions

When a tool accepts a URL reference:
- absolute forum URL is valid
- forum-relative path is valid
- internal XenForo route variants such as `latest`, `unread` and paged URLs are valid where relevant

### `list_forums`

Purpose:
- enumerate categories and forums for navigation

Input:
- optional `base_url`

Output:
- categories
- forums per category
- canonical forum URLs

Notes:
- this is the entrypoint for discovery

Result fields:
- `categories[].title`
- `categories[].url`
- `categories[].description`
- `categories[].forums[].title`
- `categories[].forums[].url`

### `list_threads`

Purpose:
- list threads in a given forum with pagination

Input:
- `forum_url`
- optional `page`

Output:
- forum title
- current page
- thread summaries
- next-page URL if available

Each thread summary should include:
- title
- canonical URL
- author
- started-at timestamp string
- replies
- views
- last-post-at timestamp string
- last-poster

Result fields:
- `forum_title`
- `forum_url`
- `page`
- `threads[]`
- `next_page_url`

### `read_thread`

Purpose:
- read a full thread, not just one page

Input:
- `thread_url`

Output:
- thread title
- canonical thread URL
- number of pages read
- all posts across all pages

Each post should include:
- post number
- canonical post URL
- author
- posted-at timestamp string
- text content
- images

Each image entry should include:
- `url`
- `preview_url`
- `alt`
- `attachment_url`

Result fields:
- `thread_url`
- `title`
- `posts[]`
- `pages_read`

Important:
- this tool must follow XenForo pagination until the complete thread has been collected
- partial page reads are not sufficient for the intended LLM use case

### `search_threads`

Purpose:
- search thread titles as the authenticated user

Input:
- `query`
- optional `page`

Output:
- `search_type=threads`
- query
- current page
- search results
- next-page URL if available

Each result should include:
- title
- canonical URL
- snippet

Current implemented result fields:
- `search_type`
- `query`
- `page`
- `results[].title`
- `results[].url`
- `results[].snippet`
- `next_page_url`

### `search_posts`

Purpose:
- search post contents as the authenticated user

Input:
- `query`
- optional `page`

Output:
- `search_type=posts`
- query
- current page
- search results
- next-page URL if available

Each result should include:
- title
- canonical URL
- snippet

Current implemented result fields:
- `search_type`
- `query`
- `page`
- `results[].title`
- `results[].url`
- `results[].snippet`
- `next_page_url`

### `follow_link`

Purpose:
- resolve an internal forum URL into a canonical object

Input:
- `url`

Output:
- `type`
- `canonical_url`
- object-specific metadata

Expected types:
- `forum`
- `thread`
- `post`
- `attachment`
- `image`
- `external`
- `unknown`

Examples:

For a thread link:
- canonical thread URL
- optional unread/latest normalization result

For a post link:
- canonical post URL
- parent thread URL
- post number

For a forum link:
- canonical forum URL
- forum title if resolved

For an attachment/image link:
- attachment URL
- preview URL if available
- full image URL if available

Why this tool matters:
- XenForo emits many link variants such as `unread`, `latest`, attachment pages, post anchors and forum-relative URLs
- callers should not reimplement URL normalization in each tool

Current implemented result fields:
- `input_url`
- `type`
- `canonical_url`
- `resolved_url`
- `thread_url`
- `post_url`
- `forum_url`
- `attachment_url`
- `image_url`
- `content_type`

### `get_image`

Purpose:
- expose image metadata and both preview and full-size targets when available

Input:
- `url`

Output:
- canonical image or attachment URL
- thumbnail URL
- full image URL
- alt text
- source post URL if known
- source thread URL if known

Notes:
- `read_thread` should return image references inline
- `get_image` is the dedicated resolver for image-oriented downstream clients

Current implemented result fields:
- `input_url`
- `canonical_url`
- `attachment_url`
- `thumbnail_url`
- `full_image_url`
- `content_type`

### `read_profile`

Purpose:
- read a public member profile

Input:
- `user_url`

Output:
- canonical user URL
- display name
- profile headline or title if present
- profile fields that are publicly visible
- links to public activity views when available

Notes:
- only public profile data is in scope

Current implemented result fields:
- `user_url`
- `display_name`
- `user_title`
- `joined_at`
- `last_activity`
- `post_count`
- `reaction_score`
- `recent_content_url`
- `about_url`
- `all_content_url`
- `all_threads_url`

### `list_user_posts`

Purpose:
- enumerate a user’s public forum posts

Input:
- `user_url`
- optional `page`

Output:
- canonical user URL
- current page
- public post summaries
- next-page URL if available

Each post summary should include:
- post URL
- parent thread URL
- parent thread title
- posted-at timestamp string
- snippet

Current implemented result fields:
- `user_url`
- `page`
- `posts[].title`
- `posts[].post_url`
- `posts[].thread_url`
- `posts[].thread_title`
- `posts[].posted_at`
- `posts[].forum_title`
- `posts[].forum_url`
- `posts[].snippet`
- `next_page_url`

### `list_user_threads`

Purpose:
- enumerate public threads started by a user

Input:
- `user_url`
- optional `page`

Output:
- canonical user URL
- current page
- thread summaries
- next-page URL if available

Current implemented result fields:
- `user_url`
- `page`
- `threads[].title`
- `threads[].url`
- `threads[].snippet`
- `threads[].posted_at`
- `next_page_url`

### `list_my_threads`

Purpose:
- list threads started by the authenticated user

Input:
- optional `page`

Output:
- thread summaries
- next-page URL if available

Notes:
- this is a convenience wrapper around the authenticated user’s public content
- currently backed by XenForo route `/find-threads/started`

### `list_threads_i_participated`

Purpose:
- list threads where the authenticated user has posted

Input:
- optional `page`

Output:
- thread summaries
- next-page URL if available

Notes:
- this should map to XenForo’s public “threads with your posts” or equivalent frontend views when available
- if the forum has no dedicated public view, the implementation may fall back to public user activity pages plus normalization
- currently backed by XenForo route `/find-threads/contributed`

Private messages are explicitly out of scope:
- no reading conversations
- no listing conversations
- no sending private messages

Posting is explicitly out of scope:
- no creating threads
- no replying to threads
- no editing posts
- no deleting posts

## Architecture

The PoC proved the forum behavior. The next phase should split the monolith into focused packages.

Planned structure:

```text
xf-cli/
├── DESIGN.md
├── go.mod
├── go.sum
├── main.go
├── auth/
│   ├── client.go
│   ├── login.go
│   ├── session.go
│   └── credentials.go
├── scraper/
│   ├── forums.go
│   ├── threads.go
│   ├── posts.go
│   ├── search.go
│   ├── links.go
│   └── images.go
├── mcp/
│   ├── server.go
│   ├── tools.go
│   └── types.go
└── internal/
    └── xfhtml/
        ├── fetch.go
        ├── urls.go
        └── parse.go
```

## Component Responsibilities

### `auth`

Responsible for:
- username/password sourcing
- optional session persistence later
- browser-style login flow
- CSRF token extraction
- authenticated `http.Client`
- login verification

Credential priority should be:
1. explicit CLI or tool-supplied values
2. real environment variables such as `XF_USERNAME` and `XF_PASSWORD`
3. interactive prompt when running locally

`.env` policy:
- do not load automatically
- users may explicitly `source .env`

### `scraper`

Responsible for:
- parsing XenForo HTML into typed Go structures
- pagination traversal
- canonical URL normalization
- image extraction
- public member activity extraction

This layer should not know anything about MCP.

### `mcp`

Responsible for:
- stdio transport
- JSON-RPC 2.0
- tool registration
- argument validation
- mapping tool calls to auth and scraper operations

## Session Model

Initial version:
- login per process start
- keep cookies in an in-memory jar
- fetch fresh `_xfToken` after login

Later version:
- persist session cookies and metadata to a config file
- revalidate session on startup
- relogin when expired

Suggested persisted fields:
- `base_url`
- username
- cookies
- last verified timestamp
- last `_xfToken`

## HTML Parsing Strategy

Use established libraries, not hand-rolled regex parsing.

Chosen stack:
- `goquery` for DOM querying
- standard `net/http` and `cookiejar` for requests and cookie handling

Why:
- already validated in the PoC
- enough control for XenForo’s server-rendered HTML
- simple and maintainable

Do not:
- parse core page structure with regex

## URL Canonicalization Rules

The scraper must normalize XenForo URLs aggressively.

Examples:
- relative links to absolute URLs
- thread URLs with `/unread` to canonical thread URL plus state
- thread URLs with `/latest` to canonical thread URL plus state
- post fragment and post permalink variants to canonical post URL
- attachment pages to canonical attachment URL
- image preview URLs to full attachment/image URLs when determinable

This logic should live in one place, not inside every tool.

## Pagination Rules

### Forums and searches

- return one page at a time
- expose `next_page_url`
- callers decide whether to continue

### User activity

- return one page at a time
- expose `next_page_url`
- normalize threads and post links back into canonical forum objects

### Threads

- `read_thread` must traverse all pages by default
- stop only when no next page is found or a URL loop is detected

Safeguards:
- maintain a visited-URL set
- fail clearly on repeated-page loops

## Image Support

Image support is part of the read path, not an afterthought.

### In `read_thread`

Each post should expose:
- inline linked images
- XenForo attachment preview images
- external image URLs

Each image object should aim to provide:
- preview URL
- full image or attachment URL
- alt text

### In `get_image`

Resolve:
- attachment page URLs
- preview thumbnails
- direct image links

Return both:
- thumbnail URL
- full-size URL

If only one exists, return that fact explicitly.

## Error Semantics

The server should return structured, user-readable errors.

Important classes:
- login failed
- session expired
- CSRF token missing
- page structure changed
- forum or thread not found
- insufficient permissions
- rate-limited or blocked
- unsupported link type

Errors should preserve the forum’s German messages where useful, but also add a stable machine-readable code.

## Observability

Verbose logging should remain available because it was essential for the PoC.

Log when verbose:
- request method and URL
- form fields with masked password
- response status
- response headers
- truncated body
- cookies seen/set
- pagination traversal decisions
- canonicalization decisions for `follow_link`

## Risks

### XenForo theme or markup changes

Risk:
- selectors stop matching

Mitigation:
- centralize selectors
- prefer semantic containers over brittle deep selectors
- add HTML fixture tests

### Search behavior and CSRF handling

Risk:
- search flow differs between quick search and advanced search

Mitigation:
- drive `search_threads` and `search_posts` through the real frontend search form flow
- cover the result parsing with fixtures from `rc-network.de`

### Large threads

Risk:
- reading entire threads can be slow and memory-heavy

Mitigation:
- stream page-by-page internally
- consider later output limits or chunking in MCP
- keep full-thread semantics, but make implementation memory-conscious

### External images

Risk:
- image URLs may point off-site and disappear or block hotlinking

Mitigation:
- preserve original URLs
- distinguish local XenForo attachments from external images

## Testing Strategy

Add tests before or during the refactor from `main.go` to packages.

Test layers:

- unit tests for URL normalization
- unit tests for HTML parsing on saved fixtures
- session/login tests around token extraction and login-state detection
- integration smoke tests for the live target, only when credentials are explicitly available

Recommended fixture set:
- login page
- authenticated home page
- forum overview
- forum thread list
- thread page with pagination
- thread page with multiple images
- search page
- search result page
- public member profile page
- public member posts page
- public member threads page

## Implementation Order After This Design

1. Extract auth code from `main.go` into `auth/`
2. Extract scraper parsers into `scraper/`
3. Add fixture-based tests for parsing and normalization
4. Introduce MCP stdio server and register tools
5. Implement public user-profile and user-activity tools
6. Implement `follow_link`
7. Implement `get_image`
8. Add session persistence

## Summary

The PoC already proved the core viability:
- browser login works
- frontend scraping works
- full thread traversal works
- images can be extracted
- `search_threads` works
- `search_posts` works
- REST API without `XF-Api-Key` is not usable for this project

The next correct move is not more PoC expansion in `main.go`.
The next correct move is:
- refactor into packages
- preserve the already validated behavior
- surface the validated read-paths as MCP tools
- keep the server strictly read-only
