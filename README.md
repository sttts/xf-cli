# xf-cli

`xf-cli` is a read-only XenForo frontend client and MCP server in Go.

It logs into a XenForo 2.x forum as a normal user, without `XF-Api-Key`, and exposes forum content through:
- a CLI with MCP-aligned subcommands
- an MCP stdio server for Claude, Codex, and other MCP clients

The current validation target is:
- `https://www.rc-network.de`

## Scope

Implemented:
- browser-style login with username/password
- verified session reuse from `~/.config/xf-cli/session.json`
- forum listing
- thread listing
- full thread reading across all pages
- thread-title search
- post-content search
- public profile reading
- user post and thread listing
- "my threads" and "threads I participated in"
- link normalization
- image and attachment resolution
- MCP over `stdin` / `stdout`

Not in scope:
- writing posts
- creating threads
- edits or deletes
- private messages
- XenForo REST API usage
- automatic `.env` loading

## Why frontend login

`rc-network.de` exposes the XenForo REST API, but it still requires `XF-Api-Key`.
Normal browser session cookies do not unlock `/api/...`.

So this project uses the authenticated frontend and parses HTML instead.

## Build

```bash
go build ./...
```

## Authentication

Priority order:
1. `-u` / `-p`
2. `XF_USERNAME` / `XF_PASSWORD`
3. interactive prompt

Example:

```bash
export XF_USERNAME=sttts
export XF_PASSWORD='...'
go run . list_forums
```

`.env` is intentionally not auto-loaded.

## Session persistence

Successful logins are stored in:

```text
~/.config/xf-cli/session.json
```

On later runs, `xf-cli` tries to:
1. load the stored session
2. verify it against the forum frontend
3. reuse it if still valid
4. fall back to a fresh login if it expired

## CLI

Top-level commands:
- `login`
- `list_forums`
- `list_threads`
- `read_thread`
- `search_threads`
- `search_posts`
- `read_profile`
- `list_user_posts`
- `list_user_threads`
- `list_my_threads`
- `list_threads_i_participated`
- `follow_link`
- `get_image`
- `mcp`

Examples:

```bash
go run . login
go run . list_forums --json
go run . list_threads /forums/flugmodellbau-allgemein.31/ --limit=50
go run . read_thread /threads/eure-sch%C3%B6nsten-modelle.144946/ --json
go run . search_threads Fouga --limit=20
go run . search_posts Fouga --limit=20
go run . read_profile /members/sttts.31018/
go run . follow_link /threads/eure-sch%C3%B6nsten-modelle.144946/latest
go run . get_image /attachments/piper-tc-jpg.9277151/
```

### Logical paging

List and search commands use logical paging:
- `--limit=100` is the CLI default
- `--limit=0` means: collect all pages
- if `--page` is unset, pages are collected until at least `limit` results are present
- responses include `next_page` and `next_page_url`
- pass `--page=<next_page>` to continue

`read_thread` is different:
- it always reads the complete thread
- there is no paging flag for it

## Debug logging

`-v` is a counter:
- `-v`: request line
- `-vv`: request form data, status, headers, cookies
- `-vvv`: response body as well

Example:

```bash
go run . -v search_posts Fouga --limit=5
go run . -vv list_threads /forums/flugmodellbau-allgemein.31/ --limit=5
go run . -vvv read_thread /threads/eure-sch%C3%B6nsten-modelle.144946/
```

## MCP mode

Run the stdio MCP server with:

```bash
go run . mcp
```

The server speaks MCP over `stdin` / `stdout`. It does not expose an HTTP transport.

Current MCP tools:
- `list_forums`
- `list_threads`
- `read_thread`
- `search_threads`
- `search_posts`
- `read_profile`
- `list_user_posts`
- `list_user_threads`
- `list_my_threads`
- `list_threads_i_participated`
- `follow_link`
- `get_image`

## Tests

Unit and live tests:

```bash
make test
```

Live tests run against `rc-network.de` and require:

```bash
export XF_USERNAME=...
export XF_PASSWORD=...
```

The suite verifies:
- login
- session reuse
- forum and thread reads
- search tools
- user/profile tools
- link and image tools
- MCP tool registration and MCP tool calls

## Repository

GitHub repository:

```text
git@github.com:sttts/xf-cli.git
```
