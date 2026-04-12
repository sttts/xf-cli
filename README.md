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

## Install

Install the CLI into your Go bin directory:

```bash
go install github.com/sttts/xf-cli@latest
```

Then use it directly:

```bash
xf-cli --help
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
xf-cli list_forums
```

`.env` is intentionally not auto-loaded.

### `login` persists the session

This command does not only test credentials:

```bash
xf-cli login
```

A successful `login` writes the authenticated session to:

```text
~/.config/xf-cli/session.json
```

That stored session is then reused by:
- later CLI commands
- `xf-cli mcp`

So the normal setup flow is:
1. run `xf-cli login`
2. confirm that `session.json` exists
3. use the other CLI commands or start the MCP server

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

This applies to both:
- CLI commands such as `list_threads`
- MCP mode via `xf-cli mcp`

If no valid stored session exists, `xf-cli` needs credentials again.

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
xf-cli login
xf-cli list_forums --json
xf-cli list_threads /forums/flugmodellbau-allgemein.31/ --limit=50
xf-cli read_thread /threads/eure-sch%C3%B6nsten-modelle.144946/ --json
xf-cli search_threads Fouga --limit=20
xf-cli search_posts Fouga --limit=20
xf-cli read_profile /members/sttts.31018/
xf-cli follow_link /threads/eure-sch%C3%B6nsten-modelle.144946/latest
xf-cli get_image /attachments/piper-tc-jpg.9277151/
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
xf-cli -v search_posts Fouga --limit=5
xf-cli -vv list_threads /forums/flugmodellbau-allgemein.31/ --limit=5
xf-cli -vvv read_thread /threads/eure-sch%C3%B6nsten-modelle.144946/
```

## MCP mode

Run the stdio MCP server with:

```bash
xf-cli mcp
```

The server speaks MCP over `stdin` / `stdout`. It does not expose an HTTP transport.

### MCP authentication

The MCP server uses the same auth/session logic as the CLI.

Recommended setup:

1. create a session once:

```bash
xf-cli login
```

2. start the MCP server:

```bash
xf-cli mcp
```

If the stored session is still valid, MCP starts without prompting.

If the stored session is missing or expired, MCP needs credentials from:
- `XF_USERNAME`
- `XF_PASSWORD`

Example:

```bash
XF_USERNAME=sttts XF_PASSWORD='...' xf-cli mcp
```

Important:
- `.env` is not auto-loaded
- MCP must not depend on interactive prompts in normal client setup
- for reliable MCP startup, either create the session first with `login`, or pass `XF_USERNAME` / `XF_PASSWORD` in the environment of the MCP process

### Claude Desktop

`xf-cli` is a local stdio MCP server. In Claude Desktop, configure it as a command-based server and pass credentials through environment variables if you do not want to rely on the persisted session.

Typical local setup:

```json
{
  "mcpServers": {
    "xf-cli": {
      "command": "/absolute/path/to/xf-cli",
      "args": ["mcp"],
      "env": {
        "XF_USERNAME": "your-user",
        "XF_PASSWORD": "your-password"
      }
    }
  }
}
```

If you already created `~/.config/xf-cli/session.json` with `xf-cli login`, the `env` block can often be omitted until the session expires.

### Claude

For Claude clients that support local MCP stdio servers, use the same pattern:
- launch `xf-cli mcp`
- keep it on `stdin` / `stdout`
- provide `XF_USERNAME` / `XF_PASSWORD` in the server environment if no persisted session is available

The exact UI differs by Claude surface, but the important part is the same command:

```bash
/absolute/path/to/xf-cli mcp
```

### Codex

Codex supports adding a local stdio MCP server directly.

Example:

```bash
codex mcp add xf-cli --env XF_USERNAME=your-user --env XF_PASSWORD=your-password -- /absolute/path/to/xf-cli mcp
```

Then verify it with:

```bash
codex mcp list
```

If you want to rely on the persisted session instead, first run:

```bash
xf-cli login
```

and then add the MCP server without the credential env vars.

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
