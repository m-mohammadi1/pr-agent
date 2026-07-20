# pr-agent

A portable Go CLI that bridges GitHub PR review comments and AI agents. Fetch unresolved review threads, get fix context with diff hunks, reply to comments, and resolve threads — all via structured JSON on stdout.

## Install

Install to your `GOPATH/bin` (so `pr-agent` is on your PATH):

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

### `context` — main agent entry point

Returns an actionable fix queue with file paths, lines, comment bodies, diff hunks, and full conversation history per thread.

Includes:
- **inline_review** — line-level review threads (paginated, full reply chain)
- **issue_comment** — top-level PR conversation comments
- **review_body** — submitted review summary bodies

```bash
pr-agent context --repo owner/repo --pr 42
pr-agent context --repo owner/repo --pr 42 --unresolved=false
pr-agent context --repo owner/repo --pr 42 --inline-only
pr-agent context --repo owner/repo --pr 42 --no-conversation
```

### `list` — raw review threads and conversation comments

```bash
pr-agent list --repo owner/repo --pr 42
pr-agent list --repo owner/repo --pr 42 --unresolved
pr-agent list --repo owner/repo --pr 42 --inline-only
```

### `status` — thread counts

```bash
pr-agent status --repo owner/repo --pr 42
```

### `reply` — reply to a comment

```bash
# Inline review thread (default)
pr-agent reply --repo owner/repo --pr 42 --comment-id 123456 --body "Fixed in abc123"

# PR conversation or review-body comment
pr-agent reply --repo owner/repo --pr 42 --comment-id 789 --kind issue_comment --body "Addressed"
pr-agent reply --repo owner/repo --pr 42 --comment-id 456 --kind review_body --body "Thanks, fixed"
```

Use the numeric `comment_id` from `context` or `list` output. For inline threads, replies are threaded. For conversation/review-body comments, a new PR comment is posted.

### `resolve` — resolve an inline review thread

```bash
pr-agent resolve --thread-id PRRT_abc123
```

Only applies to **inline_review** threads (`thread_id` starting with `PRRT_`). Idempotent — resolving an already-resolved thread succeeds.

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
      "diff_hunk": "@@ -40,7 +40,7 @@ func foo() {",
      "comments": [
        {
          "id": "1234567890",
          "body": "Consider handling the error here.",
          "author": "coderabbit[bot]",
          "created_at": "2026-07-20T08:00:00Z",
          "diff_hunk": "@@ -40,7 +40,7 @@ func foo() {"
        },
        {
          "id": "1234567900",
          "body": "Still needs a nil check.",
          "author": "coderabbit[bot]",
          "created_at": "2026-07-20T09:00:00Z"
        }
      ]
    }
  ],
  "summary": {
    "total": 5,
    "unresolved": 2,
    "resolved": 3,
    "outdated": 0,
    "inline": 3,
    "issue": 1,
    "review_bodies": 1
  }
}
```

## Design

- **REST** (`go-github`): issue comments, review bodies, inline replies
- **GraphQL**: list inline review threads with resolve state, paginated comment history, resolve threads
- **stdout = JSON**, stderr = logs/errors
- No webhooks — agents poll `context` when needed

## License

MIT

## Releasing

Push a version tag to trigger the release workflow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions builds binaries for linux/darwin/windows (amd64 + arm64) and publishes them to the Releases page.
