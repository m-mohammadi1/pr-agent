package guide

// AgentGuide is the full workflow and reference doc for AI agents using pr-agent
// via MCP tools or the CLI. Shared by get_agent_guide (MCP) and pr-agent --help.
const AgentGuide = `pr-agent — GitHub PR review bridge for AI agents

PURPOSE
  Fetch PR review feedback from GitHub (inline threads, conversation comments,
  review bodies), fix code locally, then reply and resolve threads on GitHub.
  MCP tools return JSON text content. CLI commands return JSON on stdout.

AUTH (run once before other tools/commands)
  MCP:  auth_status                    Check authenticated user
  CLI:  pr-agent auth login           Interactive one-time setup
        pr-agent auth status
        pr-agent auth logout

  Token resolution: GITHUB_TOKEN → GH_TOKEN → TOKEN → ~/.config/pr-agent/config.json

AGENT WORKFLOW
  0. MCP only: call get_agent_guide if unsure (this document).

  1. Fetch unresolved review items:
     MCP:  get_pr_context { repo, pr, unresolved: true }
     CLI:  pr-agent context --repo OWNER/REPO --pr N

     Each item in "items" is one actionable thread. Read file, line, body,
     comments[] (full chain), and diff_hunk to understand what to fix.

  2. Fix code locally and commit (pr-agent does not commit or push).

  3. For each addressed item:
     MCP:  reply_to_comment { repo, pr, comment_id, body, kind? }
     CLI:  pr-agent reply --repo OWNER/REPO --pr N --comment-id ID --body "Fixed in SHA"

     MCP:  resolve_thread { thread_id }     (inline_review only)
     CLI:  pr-agent resolve --thread-id PRRT_...

  4. Verify all addressed:
     MCP:  pr_status { repo, pr }
     CLI:  pr-agent status --repo OWNER/REPO --pr N
     Confirm summary.unresolved is 0 (or only non-actionable items remain).

COMMENT KINDS (field "kind" in JSON output)
  inline_review   Line-level review thread. thread_id prefix: PRRT_
                  Reply: threaded (kind inline_review). Resolve: yes.
  issue_comment   Top-level PR conversation comment. thread_id prefix: IC_
                  Reply: new PR comment (kind issue_comment). Resolve: no.
  review_body     Submitted review summary. thread_id prefix: PRR_
                  Reply: new PR comment (kind review_body). Resolve: no.

KEY FIELDS IN get_pr_context OUTPUT
  items[].thread_id     GraphQL or synthetic ID; use with resolve_thread (inline only)
  items[].comment_id    Latest comment database ID; use with reply_to_comment
  items[].comments[]    Full conversation chronologically (multi-level replies)
  items[].body          Latest comment text (convenience; same as last comments[])
  items[].file,line     File location for inline_review
  items[].is_outdated   true if diff line moved; still worth addressing
  items[].is_resolved   true if thread already resolved (inline only)
  summary               Counts: total, unresolved, resolved, outdated, inline, issue, review_bodies

MCP TOOLS
  get_agent_guide    This guide — call first if unsure
  get_pr_context     Primary entry: fix queue with full comment history
  list_pr_threads    Raw threads with nested comments (prefer get_pr_context)
  pr_status          Aggregate counts without full payloads
  reply_to_comment   Post a reply after fixing
  resolve_thread     Mark inline review thread resolved (PRRT_ only)
  unresolve_thread   Re-open a resolved inline review thread (PRRT_ only)
  auth_status        Show authenticated GitHub user

CLI COMMANDS (shell alternative to MCP)
  context     Same as get_pr_context
  list        Same as list_pr_threads
  status      Same as pr_status
  reply       Same as reply_to_comment
  resolve     Same as resolve_thread
  unresolve   Same as unresolve_thread
  auth        One-time GitHub authentication

NOTES FOR AGENTS
  - Process every item where is_resolved is false.
  - Read comments[] for full back-and-forth, not just body.
  - is_outdated true means the line moved; read body and file anyway.
  - Use comment_id for reply; thread_id for resolve.
  - Mention commit SHA in reply body so reviewers can verify the fix.
  - resolve_thread only works for inline_review (thread_id starting with PRRT_).
  - unresolve_thread re-opens a resolved inline thread (thread_id starting with PRRT_).

CLI EXIT CODES
  0  Success — parse stdout as JSON
  1  Usage or auth error
  2  GitHub API error — token may lack repo access or rate-limited`
