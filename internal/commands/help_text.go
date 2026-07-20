package commands

// Agent-oriented help text for pr-agent commands.
// Written for AI agents that invoke this CLI as a shell tool.

const rootLong = `pr-agent — GitHub PR review bridge for AI agents

PURPOSE
  Fetch PR review feedback from GitHub (inline threads, conversation comments,
  review bodies), fix code locally, then reply and resolve threads on GitHub.
  All successful command output is JSON on stdout. Errors go to stderr.

AUTH (run once before other commands)
  pr-agent auth login          Interactive one-time setup; stores token globally
  pr-agent auth status         Verify authentication
  pr-agent auth logout         Remove stored token

  Token resolution: GITHUB_TOKEN → GH_TOKEN → TOKEN → ~/.config/pr-agent/config.json

AGENT WORKFLOW
  1. pr-agent context --repo OWNER/REPO --pr N > review.json
     Read review.json. Each item in "items" is one actionable thread.
     Use "file", "line", "body", "comments" (full chain), "diff_hunk" to fix code.

  2. Fix code locally and commit (pr-agent does not commit or push).

  3. For each addressed item:
     pr-agent reply --repo OWNER/REPO --pr N --comment-id ID --body "Fixed in SHA"
     pr-agent resolve --thread-id THREAD_ID    (inline_review only)

  4. pr-agent status --repo OWNER/REPO --pr N   Verify unresolved count is 0

COMMENT KINDS (field "kind" in JSON output)
  inline_review   Line-level review thread. thread_id prefix: PRRT_
                  Reply: threaded. Resolve: yes (--thread-id).
  issue_comment   Top-level PR conversation comment. thread_id prefix: IC_
                  Reply: new PR comment (--kind issue_comment). Resolve: no.
  review_body     Submitted review summary. thread_id prefix: PRR_
                  Reply: new PR comment (--kind review_body). Resolve: no.

KEY FIELDS IN context OUTPUT
  items[].thread_id     GraphQL or synthetic ID; use with resolve (inline only)
  items[].comment_id    Latest comment database ID; use with reply
  items[].comments[]    Full conversation chronologically (multi-level replies)
  items[].body          Latest comment text (convenience; same as last comments[])
  items[].file,line     File location for inline_review
  items[].is_outdated   true if diff line moved; still worth addressing
  items[].is_resolved   true if thread already resolved (inline only)
  summary               Counts: total, unresolved, resolved, outdated, inline, issue, review_bodies

EXIT CODES
  0  Success — parse stdout as JSON
  1  Usage or auth error — read stderr; run pr-agent auth login if needed
  2  GitHub API error — token may lack repo access or rate-limited

COMMANDS
  context   Primary entry: agent fix-queue with full comment history
  list      Raw thread data (same sources as context, different shape)
  status    Quick counts without full item payloads
  reply     Post a reply after fixing
  resolve   Mark inline review thread resolved
  auth      One-time GitHub authentication`

const contextLong = `Build an agent-ready fix queue for a pull request.

WHEN TO USE
  Start here. This is the main command agents should call to understand what
  needs fixing on a PR. Prefer context over list when you need file/line/diff
  context in a flat item array.

OUTPUT (JSON on stdout)
  {
    "repo": "owner/repo",
    "pr": 42,
    "items": [{
      "thread_id": "PRRT_...",
      "kind": "inline_review",
      "file": "path/to/file.go",
      "line": 42,
      "body": "latest comment text",
      "author": "bot-name",
      "comment_id": "1234567890",
      "diff_hunk": "@@ ...",
      "is_resolved": false,
      "is_outdated": false,
      "comments": [
        {"id": "...", "body": "...", "author": "...", "created_at": "..."}
      ]
    }],
    "summary": {"total": 3, "unresolved": 1, "inline": 2, "issue": 1, ...}
  }

FLAGS
  --unresolved (default true)   Skip resolved inline threads; conversation comments always included
  --inline-only                 Only inline review threads (skip issue/review_body)
  --no-conversation             Exclude issue_comment and review_body
  --include-issue               Include PR conversation comments (default true)
  --include-review-body         Include review summary bodies (default true)

NOTES FOR AGENTS
  - Process every item where is_resolved is false.
  - Read comments[] for full back-and-forth, not just body.
  - is_outdated true means the line moved; read body and file anyway.
  - Use comment_id from the item for reply; thread_id for resolve.`

const listLong = `List all review threads and conversation comments for a PR.

WHEN TO USE
  When you need the raw thread structure with nested comments[] per thread.
  Use context instead if you want a flat fix-queue optimized for agents.

OUTPUT (JSON on stdout)
  {
    "repo": "owner/repo",
    "pr": 42,
    "threads": [{
      "thread_id": "PRRT_...",
      "kind": "inline_review",
      "file": "...", "line": 42,
      "is_resolved": false, "is_outdated": false,
      "comments": [{"id": "...", "body": "...", "author": "...", "created_at": "..."}]
    }]
  }

FLAGS
  Same fetch flags as context. Default --unresolved is false (returns all threads).`

const statusLong = `Return aggregate comment counts for a PR without full payloads.

WHEN TO USE
  Quick check after fixing/replying. Verify unresolved count reached zero.
  Cheaper than context when you only need counts.

OUTPUT (JSON on stdout)
  {
    "repo": "owner/repo",
    "pr": 42,
    "summary": {
      "total": 5, "unresolved": 2, "resolved": 3, "outdated": 0,
      "inline": 3, "issue": 1, "review_bodies": 1
    }
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
  --comment-id   Numeric database ID from context/list output (required)
  --body         Reply text (required)
  --kind         inline_review | issue_comment | review_body (default inline_review)

NOTES FOR AGENTS
  - Use the comment_id from the item you addressed (usually the latest in comments[]).
  - Mention the commit SHA in --body so reviewers can verify the fix.
  - For inline_review, --comment-id can be any comment in the thread.`

const resolveLong = `Mark an inline review thread as resolved on GitHub.

WHEN TO USE
  After fixing and replying to an inline_review item. Idempotent.

LIMITATIONS
  Only works for inline_review threads (thread_id starting with PRRT_).
  issue_comment and review_body cannot be resolved via this command.

OUTPUT (JSON on stdout)
  {"thread_id": "PRRT_...", "is_resolved": true}

FLAGS
  --thread-id   GraphQL thread ID from context/list (required, prefix PRRT_)`

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
