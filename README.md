# pr-agent

CLI (+ MCP server) that bridges GitHub PR review comments and AI agents — and works the same way for humans in a terminal. Fetch review feedback as JSON, fix code locally, then reply and resolve threads.

## How to use

```text
info      →  title + description
reviews   →  INDEX: which reviews / orphans need work?
review    →  DETAIL: load one id fully (items, comments, diffs)
(fix)    →  edit + commit in your repo
reply     →  comment on the thread
resolve   →  close the inline item (PRRT_ only)
reviews   →  confirm items_unresolved is 0
```

| Command | Role |
|---------|------|
| `reviews` | **Index** of the PR — counts and previews only |
| `review --id …` | **Detail** for one review or orphan |

```bash
pr-agent auth login
pr-agent info    --repo owner/repo --pr 42
pr-agent reviews --repo owner/repo --pr 42
pr-agent review  --repo owner/repo --pr 42 --id PRR_123
pr-agent reply   --repo owner/repo --pr 42 --comment-id 456 --body "Fixed in abc123"
pr-agent resolve --thread-id PRRT_...
```

Agents via MCP use the same flow: `get_pr_info` → `list_reviews` → `get_review` → `reply_to_comment` → `resolve_thread`. Run `pr-agent --help` or call `get_agent_guide` for the full guide.

## Install

Install to your `GOPATH/bin` (so `pr-agent` is on your PATH):

```bash
go install github.com/m-mohammadi1/pr-agent/cmd/pr-agent@latest
```

From a local checkout:

```bash
go install ./cmd/pr-agent
```

Or download a pre-built binary from [GitHub Releases](https://github.com/m-mohammadi1/pr-agent/releases):

```bash
# Linux amd64 example (replace VERSION, e.g. v0.1.0)
VERSION=v0.1.0
curl -LO "https://github.com/m-mohammadi1/pr-agent/releases/download/${VERSION}/pr-agent_${VERSION}_linux_amd64"
chmod +x "pr-agent_${VERSION}_linux_amd64"
sudo mv "pr-agent_${VERSION}_linux_amd64" /usr/local/bin/pr-agent
```

Or build a local binary:

```bash
go build -o bin/pr-agent ./cmd/pr-agent
# or, using the docker-based Makefile:
make build
```

## Auth

Authenticate once. The token is stored in your user config dir
(`~/.config/pr-agent/config.json`, mode `0600`) and reused everywhere — no need to
set a token per terminal or per project.

```bash
pr-agent auth login
```

The interactive flow will:

1. Offer to reuse your GitHub CLI token (`gh auth token`) if available.
2. Otherwise prompt you to paste a token (input is hidden).
3. Validate the token and save it globally.

Non-interactive options (for scripts/CI):

```bash
pr-agent auth login --from-gh          # use gh auth token, no prompts
pr-agent auth login --token ghp_xxx    # provide a token directly
```

Check or clear auth:

```bash
pr-agent auth status
pr-agent auth logout
```

Token resolution order (first match wins):

1. `GITHUB_TOKEN`
2. `GH_TOKEN`
3. `TOKEN`
4. stored config (`pr-agent auth login`)

Env vars override the stored token, which is handy in CI.

Required scopes for a fine-grained PAT:

- Repository access: the repos/org you target
- Pull requests: Read and write
- Contents: Read (for future file context)

## Commands

All commands write JSON to stdout. Logs and errors go to stderr.

### `auth` — one-time setup

```bash
pr-agent auth login
pr-agent auth status
pr-agent auth logout
```

See [Auth](#auth) above for details.

### `reviews` — index (plural)

Lists reviews + orphans with unresolved/resolved **counts** and short previews. Use this to pick an id.

```bash
pr-agent reviews --repo owner/repo --pr 42
```

### `review` — detail (singular)

Loads **one** review or orphan by `--id` with full body, nested items, comments, and diff hunks.

```bash
pr-agent review --repo owner/repo --pr 42 --id PRR_123
pr-agent review --repo owner/repo --pr 42 --id IC_999
```

Then fix locally, reply with `items[].reply_comment_id`, and resolve with `items[].thread_id` (`PRRT_` only).

### `info` — PR title and description

```bash
pr-agent info --repo owner/repo --pr 42
```

Returns title, body (description), state, draft/merged, author, URL, and base/head refs.

### `reply` — reply to a comment

```bash
# Inline review thread (default)
pr-agent reply --repo owner/repo --pr 42 --comment-id 123456 --body "Fixed in abc123"

# PR conversation or review-body comment
pr-agent reply --repo owner/repo --pr 42 --comment-id 789 --kind issue_comment --body "Addressed"
pr-agent reply --repo owner/repo --pr 42 --comment-id 456 --kind review_body --body "Thanks, fixed"
```

Use `reply_comment_id` from `review` output. For inline threads, replies are threaded. For conversation/review-body comments, a new PR comment is posted.

### `resolve` — resolve an inline review thread

```bash
pr-agent resolve --thread-id PRRT_abc123
```

Only applies to **inline** threads (`thread_id` starting with `PRRT_`). The CLI rejects `PRR_` / `IC_` before calling GitHub. Idempotent.

### `unresolve` — unresolve an inline review thread

```bash
pr-agent unresolve --thread-id PRRT_abc123
```

Re-opens a resolved inline thread. Idempotent — unresolving an already-unresolved thread succeeds.

### `mcp` — run as an MCP server

Expose pr-agent to MCP-aware clients (Cursor, Claude Code) as native tools instead of shell commands. It speaks JSON-RPC 2.0 over stdio.

```bash
pr-agent mcp
```

Tools: `get_agent_guide`, `get_pr_info`, `list_reviews`, `get_review`, `reply_to_comment`, `resolve_thread`, `unresolve_thread`, `auth_status`.

Call `get_agent_guide` first if unsure how the workflow works.

Add to your MCP client config (e.g. `~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "pr-agent": {
      "command": "pr-agent",
      "args": ["mcp"]
    }
  }
}
```

Authentication uses the same resolution as the CLI, so run `pr-agent auth login` once first. Then, in your MCP client, you can ask things like:

> Use pr-agent to list reviews on owner/repo PR 42, load the ones with unresolved items, fix them, then reply and resolve.

The agent calls `list_reviews` → `get_review`, edits files, then `reply_to_comment` and `resolve_thread` — no manual JSON copying.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage or auth error |
| 2 | GitHub API error |

## Agent workflow

```bash
# 1. Optional: PR title/description
pr-agent info --repo myorg/myrepo --pr 42

# 2. Index reviews + orphans
pr-agent reviews --repo myorg/myrepo --pr 42

# 3. Load one review with full items
pr-agent review --repo myorg/myrepo --pr 42 --id PRR_123

# 4. Fix locally, then reply and resolve each item
pr-agent reply --repo myorg/myrepo --pr 42 --comment-id 123 --body "Fixed in commit abc123"
pr-agent resolve --thread-id PRRT_kwDO...
```

## Design

- **REST** (`go-github`): issue comments, review list/bodies, inline replies, PR metadata
- **GraphQL**: inline review threads (with parent review id), resolve/unresolve
- **Review hierarchy**: `reviews` / `list_reviews` → `review` / `get_review` → reply → resolve
- **MCP**: stdio JSON-RPC 2.0 server (stdlib only) wrapping the same operations
- **stdout = JSON**, stderr = logs/errors
- No webhooks — agents poll `reviews` when needed

## License

MIT

## Releasing

Push a version tag to trigger the release workflow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions builds binaries for linux/darwin/windows (amd64 + arm64) and publishes them to the Releases page.
