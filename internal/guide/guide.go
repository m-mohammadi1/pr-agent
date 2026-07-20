package guide

// AgentGuide is the full workflow and reference doc for AI agents and humans
// using pr-agent via MCP tools or the CLI. Shared by get_agent_guide (MCP)
// and pr-agent --help.
const AgentGuide = `pr-agent — GitHub PR review bridge for humans and AI agents

WHAT IT DOES
  Pull review feedback from GitHub, fix code in your editor, then reply and
  resolve threads on the PR. Output is always JSON on stdout (errors on stderr).

  Mental model:
    PR
     ├── review (summary comment)     ← not resolvable; may have nested items
     │    ├── item (inline thread)    ← file/line; reply + resolve
     │    └── item
     └── orphans                      ← e.g. Dependency Security Scan (reply only)

QUICK START (humans)
  1. pr-agent auth login
  2. pr-agent info    --repo OWNER/REPO --pr N     # title + description
  3. pr-agent reviews --repo OWNER/REPO --pr N     # index: what needs work?
  4. pr-agent review  --repo OWNER/REPO --pr N --id PRR_...   # load one fully
  5. Fix code locally, commit
  6. pr-agent reply   --repo OWNER/REPO --pr N --comment-id ID --body "Fixed in SHA"
  7. pr-agent resolve --thread-id PRRT_...
  8. pr-agent reviews --repo OWNER/REPO --pr N     # confirm items_unresolved is 0

QUICK START (agents / MCP)
  Same steps via tools: get_pr_info → list_reviews → get_review →
  (edit files) → reply_to_comment → resolve_thread → list_reviews.
  Call get_agent_guide first if unsure. Do not invent shell flags; use JSON tool args.

reviews vs review (important)
  reviews   INDEX of the whole PR. Lightweight: each review's preview +
            items_unresolved / items_resolved counts, plus orphans[].
            Use this to decide WHICH review (or orphan) to open next.
            CLI: pr-agent reviews --repo OWNER/REPO --pr N
            MCP: list_reviews { repo, pr }

  review    DETAIL for ONE id. Full summary body + all nested items with
            comments[], diff_hunk, file/line, reply_comment_id, thread_id.
            Use this AFTER picking an id from reviews.
            CLI: pr-agent review --repo OWNER/REPO --pr N --id PRR_...
            MCP: get_review { repo, pr, id }

AUTH (once per machine)
  CLI:  pr-agent auth login | auth status | auth logout
  MCP:  auth_status
  Token order: GITHUB_TOKEN → GH_TOKEN → TOKEN → ~/.config/pr-agent/config.json

WORKFLOW DETAIL
  1. info / get_pr_info
     Read title and description so you know what the PR claims to do.

  2. reviews / list_reviews
     Look at summary.items_unresolved and each reviews[].items_unresolved.
     Pick a review_id (PRR_...) with unresolved items, or an orphan id.

  3. review / get_review --id <id>
     For each items[] where can_resolve=true and is_resolved=false:
       - Read file, line, diff_hunk, comments[]
       - Fix code locally (pr-agent does not commit or push)
       - reply with reply_comment_id (kind inline_review)
       - resolve with thread_id (PRRT_ only)

  4. Optionally reply to the review summary using reply.comment_id
     (kind review_body). Orphans like security scans: reply only, never resolve.

  5. Run reviews again to verify items_unresolved reached 0.

ID PREFIXES
  PRR_   Review summary. can_resolve=false. Reply: --kind review_body
  PRRT_  Inline item.    can_resolve=true.  Reply: --kind inline_review. Resolve: yes
  IC_    Orphan issue comment. can_resolve=false. Reply: --kind issue_comment

  resolve / resolve_thread reject PRR_ and IC_ before calling GitHub.

COMMANDS / MCP TOOLS
  info / get_pr_info           PR title, description, state, branches, URL
  reviews / list_reviews       Index: all reviews + orphans + counts
  review / get_review          Full detail for one review or orphan (--id)
  reply / reply_to_comment     Post a reply after fixing
  resolve / resolve_thread     Resolve inline item (PRRT_ only)
  unresolve / unresolve_thread Re-open a resolved inline item
  auth / auth_status           Authentication
  mcp                          Run as MCP stdio server (JSON-RPC over stdin/stdout)
  get_agent_guide              This document (MCP only)

TIPS
  - Always reviews → review; never guess thread ids.
  - Prefer comments[] over a single body field for full conversation context.
  - is_outdated=true still needs attention; the line may have moved.
  - Mention the fix commit SHA in reply bodies.
  - Empty GitHub "reply shells" (no body, no items) are omitted from reviews.

EXIT CODES (CLI)
  0  Success — parse stdout as JSON
  1  Usage or auth error (including not-resolvable id)
  2  GitHub API error — token may lack repo access or rate-limited`
