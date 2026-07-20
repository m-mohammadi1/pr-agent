package commands

import "github.com/m-mohammadi1/pr-agent/internal/guide"

// Agent-oriented help text for pr-agent commands.
// Written for AI agents that invoke this CLI as a shell tool.

const rootLong = guide.AgentGuide

const reviewsLong = `List all reviews on a PR (index) — not the full comment details.

WHAT THIS IS
  "reviews" (plural) = the INDEX for the whole pull request.
  Use it to see which reviews exist, how many unresolved items each has, and
  which orphans (e.g. Dependency Security Scan) are on the conversation.

  For the full body + every line comment under one review, use:
    pr-agent review --repo OWNER/REPO --pr N --id PRR_...

WHEN TO USE
  First step after (optional) "info". Humans and agents both start here to
  decide what to open next. Empty reply-only PullRequestReview shells are omitted.

OUTPUT (JSON on stdout)
  {
    "repo": "owner/repo", "pr": 42,
    "summary": {"reviews": 2, "orphans": 1, "items_unresolved": 3, ...},
    "reviews": [{
      "review_id": "PRR_123", "author": "jasper", "state": "CHANGES_REQUESTED",
      "preview": "...", "items_total": 4, "items_unresolved": 2, "items_resolved": 2,
      "can_resolve": false
    }],
    "orphans": [{
      "id": "IC_999", "kind": "issue_comment", "preview": "Dependency Security Scan…",
      "can_resolve": false
    }]
  }

NEXT STEP
  pr-agent review --repo OWNER/REPO --pr N --id <review_id or orphan id>
  MCP equivalent: list_reviews → get_review`

const reviewLong = `Load ONE review or orphan in full — summary + all nested items.

WHAT THIS IS
  "review" (singular) = DETAIL for a single id from "reviews".
  Returns the review summary body and every nested inline item with comments,
  diff hunks, file/line, reply_comment_id, and thread_id.

  To list all reviews on the PR first, use:
    pr-agent reviews --repo OWNER/REPO --pr N

WHEN TO USE
  After "reviews", when you have chosen a review_id (PRR_...) or orphan id
  (IC_... / PRRT_...) and need everything required to fix and reply.

ID PREFIXES
  PRR_    Submitted review — body + items[] (each item can_resolve=true)
  IC_     Orphan issue comment — body only, items=[], can_resolve=false
  PRRT_   Single inline thread — items[0] with full chain; resolve the item

OUTPUT (JSON on stdout)
  {
    "id": "PRR_123", "kind": "review_body", "body": "...", "can_resolve": false,
    "reply": {"kind": "review_body", "comment_id": "123"},
    "items": [{
      "thread_id": "PRRT_...", "file": "src/foo.go", "line": 88,
      "can_resolve": true, "reply_comment_id": "456",
      "diff_hunk": "...", "comments": [...]
    }]
  }

THEN (per unresolved item)
  1. Fix code locally
  2. pr-agent reply --repo ... --pr N --comment-id <reply_comment_id> --body "Fixed in SHA"
  3. pr-agent resolve --thread-id <thread_id>
  Server rejects resolve on PRR_ / IC_ ids.
  MCP equivalent: get_review → reply_to_comment → resolve_thread`

const infoLong = `Get a pull request's title, description (body), and basic metadata.

WHEN TO USE
  Optional first step for humans and agents — understand what the PR claims to do
  before diving into reviews. Read-only; does not include review threads.

  Review feedback: use "reviews" (index) then "review" (detail).

OUTPUT (JSON on stdout)
  {
    "repo": "owner/repo", "pr": 42,
    "title": "...", "body": "...",
    "state": "open", "draft": false, "merged": false,
    "author": "...", "url": "https://github.com/...",
    "base_ref": "main", "head_ref": "feature-branch"
  }`

const replyLong = `Post a reply on a PR after addressing feedback.

WHEN TO USE
  After fixing code locally. Reply before or after resolve.

BEHAVIOR BY KIND
  inline_review (default)  Threaded reply in the review thread (CreateCommentInReplyTo)
  issue_comment            New top-level PR conversation comment
  review_body              New top-level PR conversation comment

OUTPUT (JSON on stdout)
  {"comment_id": "9876543210", "body": "your reply", "url": "https://github.com/..."}

FLAGS
  --comment-id   Numeric database ID from review items / reply target (required)
  --body         Reply text (required)
  --kind         inline_review | issue_comment | review_body (default inline_review)

NOTES FOR AGENTS
  - Use reply_comment_id from the item you addressed (usually the latest in comments[]).
  - Mention the commit SHA in --body so reviewers can verify the fix.
  - For inline_review, --comment-id can be any comment in the thread.`

const resolveLong = `Mark an inline review thread as resolved on GitHub.

WHEN TO USE
  After fixing and replying to an inline_review item. Idempotent.

LIMITATIONS
  Only works for inline_review threads (thread_id starting with PRRT_).
  Server rejects PRR_ (review summaries) and IC_ (issue comments) before calling GitHub.

OUTPUT (JSON on stdout)
  {"thread_id": "PRRT_...", "is_resolved": true}

FLAGS
  --thread-id   GraphQL thread ID from review items (required, prefix PRRT_)`

const unresolveLong = `Mark an inline review thread as unresolved on GitHub.

WHEN TO USE
  Re-open a thread that was resolved by mistake, or when further work is needed
  after a premature resolve. Idempotent.

LIMITATIONS
  Only works for inline_review threads (thread_id starting with PRRT_).
  Server rejects PRR_ and IC_ ids before calling GitHub.

OUTPUT (JSON on stdout)
  {"thread_id": "PRRT_...", "is_resolved": false}

FLAGS
  --thread-id   GraphQL thread ID from review items (required, prefix PRRT_)`

const authLong = `Manage GitHub authentication for pr-agent.

WHEN TO USE
  Run "auth login" once on a new machine. After that, all commands work
  without setting environment variables.

TOKEN STORAGE
  ~/.config/pr-agent/config.json (mode 0600)
  Override path with PR_AGENT_CONFIG env var.

SUBCOMMANDS
  login    Interactive setup (offers gh token or manual paste)
  status   Show authenticated user and token source
  logout   Remove stored token`

const authLoginLong = `Authenticate and store a GitHub token globally.

INTERACTIVE (default)
  1. Offers gh auth token if GitHub CLI is logged in
  2. Otherwise prompts for token paste (hidden input)
  3. Validates token against GitHub API
  4. Saves to ~/.config/pr-agent/config.json

NON-INTERACTIVE
  pr-agent auth login --from-gh
  pr-agent auth login --token ghp_xxx

REQUIRED TOKEN SCOPES (fine-grained PAT)
  Repository access: target repo or org
  Pull requests: Read and write
  Contents: Read`

const authStatusLong = `Show current authentication status.

Writes human-readable status to stderr:
  Authenticated user, token source (env or config), redacted token preview.

Exit 1 if not authenticated.`

const authLogoutLong = `Remove the stored token from ~/.config/pr-agent/config.json.

Does not affect GITHUB_TOKEN/GH_TOKEN/TOKEN env vars if set.`

const mcpLong = `Run pr-agent as an MCP (Model Context Protocol) stdio server.

WHEN TO USE
  Lets MCP-aware clients (Cursor, Claude Code) call pr-agent's capabilities
  as native, typed tools instead of shell commands. The process speaks JSON-RPC
  over stdin/stdout and runs until the client disconnects.

TOOLS EXPOSED
  get_agent_guide    Full usage guide for humans and agents — call first if unsure
  get_pr_info        PR title, description, and basic metadata
  list_reviews       INDEX: reviews + orphans with counts (pick an id next)
  get_review         DETAIL: one review or orphan by id (summary + all items)
  reply_to_comment   Post a reply after fixing
  resolve_thread     Resolve an inline review thread (PRRT_ only; server-verified)
  unresolve_thread   Re-open a resolved inline review thread (PRRT_ only)
  auth_status        Show authenticated GitHub user

AUTH
  Uses the same token resolution as the CLI: GITHUB_TOKEN, GH_TOKEN, TOKEN,
  then the stored config. Run "pr-agent auth login" once first.

CLIENT CONFIG (example ~/.cursor/mcp.json)
  {
    "mcpServers": {
      "pr-agent": { "command": "pr-agent", "args": ["mcp"] }
    }
  }

Do not print to stdout in this mode; stdout is reserved for the MCP protocol.`
