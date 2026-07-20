# pr-agent

A portable Go CLI that bridges GitHub PR review comments and AI agents. Fetch unresolved review threads, get fix context with diff hunks, reply to comments, and resolve threads — all via structured JSON on stdout.

## Install

Install to your `GOPATH/bin` (so `pr-agent` is on your PATH):

```bash
go install ./cmd/pr-agent
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

### `auth` — initialize token

Get a token from GitHub CLI (`gh auth token`) and write it into `.env`:

```bash
./bin/pr-agent auth from-gh --env-file .env --var TOKEN
```

Or provide one manually:

```bash
./bin/pr-agent auth manual --env-file .env --var TOKEN --token 'ghp_...'
```

Then in new terminals, source your env file once:

```bash
set -a; . ./.env; set +a
```

### `context` — main agent entry point

Returns an actionable fix queue with file paths, lines, comment bodies, and diff hunks.

```bash
pr-agent context --repo owner/repo --pr 42
pr-agent context --repo owner/repo --pr 42 --unresolved=false
```

### `list` — raw review threads

```bash
pr-agent list --repo owner/repo --pr 42
pr-agent list --repo owner/repo --pr 42 --unresolved
```

### `status` — thread counts

```bash
pr-agent status --repo owner/repo --pr 42
```

### `reply` — reply to an inline comment

```bash
pr-agent reply --repo owner/repo --pr 42 --comment-id 123456 --body "Fixed in abc123"
```

Use the numeric `comment_id` from `context` or `list` output (GitHub database ID).

### `resolve` — resolve a review thread

```bash
pr-agent resolve --thread-id PRRT_abc123
```

Use the `thread_id` from `context` or `list` output. Idempotent — resolving an already-resolved thread succeeds.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage or auth error |
| 2 | GitHub API error |

## Agent workflow

```bash
# 1. Get unresolved review items
pr-agent context --repo myorg/myrepo --pr 42 > review.json

# 2. Agent reads review.json, fixes code locally, commits

# 3. Reply and resolve each addressed thread
pr-agent reply --repo myorg/myrepo --pr 42 --comment-id 123 --body "Fixed in commit abc123"
pr-agent resolve --thread-id PRRT_kwDO...
```

## Example `context` output

```json
{
  "repo": "owner/repo",
  "pr": 42,
  "items": [
    {
      "thread_id": "PRRT_kwDO...",
      "kind": "inline_review",
      "file": "internal/foo.go",
      "line": 42,
      "side": "RIGHT",
      "is_resolved": false,
      "is_outdated": false,
      "body": "Consider handling the error here.",
      "author": "coderabbit[bot]",
      "comment_id": "1234567890",
      "diff_hunk": "@@ -40,7 +40,7 @@ func foo() {"
    }
  ],
  "summary": {
    "total": 3,
    "unresolved": 1,
    "resolved": 2,
    "outdated": 0
  }
}
```

## Design

- **REST** (`go-github`): post comment replies
- **GraphQL**: list review threads with resolve state, resolve threads
- **stdout = JSON**, stderr = logs/errors
- No webhooks — agents poll `context` when needed

## License

MIT
