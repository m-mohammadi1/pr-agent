package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hero/pr-agent/internal/github"
	"github.com/hero/pr-agent/internal/guide"
	"github.com/hero/pr-agent/internal/models"
)

// toolDef describes a single MCP tool and its handler.
type toolDef struct {
	name        string
	description string
	inputSchema map[string]any
	handler     func(ctx context.Context, args json.RawMessage) map[string]any
}

func buildTools() []toolDef {
	return []toolDef{
		{
			name: "get_agent_guide",
			description: "Return the full pr-agent usage guide for humans and agents: reviews vs review, " +
				"workflow, id prefixes, and CLI/MCP command map. Call this first if unsure.",
			inputSchema: object(nil, nil),
			handler:     getAgentGuide,
		},
		{
			name: "list_reviews",
			description: "INDEX of the PR: list submitted reviews with unresolved/resolved item counts, " +
				"plus orphans (e.g. Dependency Security Scan). Use this to choose which id to open next. " +
				"Then call get_review with that id for full details. Not a full comment dump.",
			inputSchema: prTargetSchema(),
			handler:     listReviews,
		},
		{
			name: "get_review",
			description: "DETAIL for ONE review or orphan id from list_reviews. Returns summary body plus all " +
				"nested items with comments, diff hunks, file/line, reply_comment_id, and thread_id. " +
				"Ids: PRR_ (review), IC_ (orphan issue comment), PRRT_ (single inline thread).",
			inputSchema: reviewIDSchema(),
			handler:     getReview,
		},
		{
			name:        "get_pr_info",
			description: "Return a pull request's title, description (body), state, author, branches, and URL. Read-only metadata.",
			inputSchema: prTargetSchema(),
			handler:     getPRInfo,
		},
		{
			name: "reply_to_comment",
			description: "Post a reply on a PR after addressing feedback. For inline_review the reply is threaded; " +
				"for issue_comment and review_body a new PR conversation comment is posted.",
			inputSchema: replySchema(),
			handler:     replyToComment,
		},
		{
			name: "resolve_thread",
			description: "Mark an inline review thread as resolved (thread_id starting with PRRT_). " +
				"Server rejects review summaries (PRR_) and issue comments (IC_). Idempotent.",
			inputSchema: resolveSchema(),
			handler:     resolveThread,
		},
		{
			name: "unresolve_thread",
			description: "Re-open a resolved inline review thread (thread_id starting with PRRT_). " +
				"Server rejects PRR_ and IC_ ids. Idempotent.",
			inputSchema: resolveSchema(),
			handler:     unresolveThread,
		},
		{
			name:        "auth_status",
			description: "Show the authenticated GitHub user. Use to verify authentication before other tools.",
			inputSchema: object(nil, nil),
			handler:     authStatus,
		},
	}
}

// --- schemas ---

func prTargetSchema() map[string]any {
	return object(map[string]any{
		"repo": prop("string", "repository in owner/name format, e.g. octocat/hello-world"),
		"pr":   prop("integer", "pull request number"),
	}, []string{"repo", "pr"})
}

func replySchema() map[string]any {
	return object(map[string]any{
		"repo":       prop("string", "repository in owner/name format"),
		"pr":         prop("integer", "pull request number"),
		"comment_id": prop("string", "numeric database id (items[].reply_comment_id or reply.comment_id from get_review)"),
		"body":       prop("string", "reply text; mention the commit SHA of the fix"),
		"kind":       prop("string", "comment kind: inline_review (default), issue_comment, or review_body"),
	}, []string{"repo", "pr", "comment_id", "body"})
}

func resolveSchema() map[string]any {
	return object(map[string]any{
		"thread_id": prop("string", "GraphQL thread id from get_review items, prefix PRRT_"),
	}, []string{"thread_id"})
}

func reviewIDSchema() map[string]any {
	return object(map[string]any{
		"repo": prop("string", "repository in owner/name format"),
		"pr":   prop("integer", "pull request number"),
		"id":   prop("string", "review or orphan id from list_reviews: PRR_..., IC_..., or PRRT_..."),
	}, []string{"repo", "pr", "id"})
}

func object(props map[string]any, required []string) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func prop(typ, desc string) map[string]any {
	return map[string]any{"type": typ, "description": desc}
}

// --- input types ---

type prTargetArgs struct {
	Repo string `json:"repo"`
	PR   int    `json:"pr"`
}

type replyArgs struct {
	Repo      string `json:"repo"`
	PR        int    `json:"pr"`
	CommentID string `json:"comment_id"`
	Body      string `json:"body"`
	Kind      string `json:"kind"`
}

type resolveArgs struct {
	ThreadID string `json:"thread_id"`
}

type reviewIDArgs struct {
	Repo string `json:"repo"`
	PR   int    `json:"pr"`
	ID   string `json:"id"`
}

// --- handlers ---

func listReviews(ctx context.Context, raw json.RawMessage) map[string]any {
	var in prTargetArgs
	if err := decode(raw, &in); err != nil {
		return toolError(err)
	}
	client, owner, name, err := clientAndRepo(ctx, in.Repo)
	if err != nil {
		return toolError(err)
	}
	result, err := client.ListReviews(ctx, owner, name, in.PR)
	if err != nil {
		return toolError(fmt.Errorf("list reviews: %w", err))
	}
	result.Repo = in.Repo
	return jsonResult(result)
}

func getReview(ctx context.Context, raw json.RawMessage) map[string]any {
	var in reviewIDArgs
	if err := decode(raw, &in); err != nil {
		return toolError(err)
	}
	if in.ID == "" {
		return toolError(fmt.Errorf("id is required"))
	}
	client, owner, name, err := clientAndRepo(ctx, in.Repo)
	if err != nil {
		return toolError(err)
	}
	result, err := client.GetReview(ctx, owner, name, in.PR, in.ID)
	if err != nil {
		return toolError(fmt.Errorf("get review: %w", err))
	}
	result.Repo = in.Repo
	return jsonResult(result)
}

func getPRInfo(ctx context.Context, raw json.RawMessage) map[string]any {
	var in prTargetArgs
	if err := decode(raw, &in); err != nil {
		return toolError(err)
	}
	client, owner, name, err := clientAndRepo(ctx, in.Repo)
	if err != nil {
		return toolError(err)
	}
	result, err := client.FetchPRInfo(ctx, owner, name, in.PR)
	if err != nil {
		return toolError(fmt.Errorf("get pr info: %w", err))
	}
	result.Repo = in.Repo
	return jsonResult(result)
}

func replyToComment(ctx context.Context, raw json.RawMessage) map[string]any {
	var in replyArgs
	if err := decode(raw, &in); err != nil {
		return toolError(err)
	}
	if in.Body == "" {
		return toolError(fmt.Errorf("body is required"))
	}
	client, owner, name, err := clientAndRepo(ctx, in.Repo)
	if err != nil {
		return toolError(err)
	}
	id, err := github.ParseCommentID(in.CommentID)
	if err != nil {
		return toolError(err)
	}
	kind, err := parseKind(in.Kind)
	if err != nil {
		return toolError(err)
	}
	result, err := client.Reply(ctx, owner, name, in.PR, kind, id, in.Body)
	if err != nil {
		return toolError(err)
	}
	return jsonResult(result)
}

func resolveThread(ctx context.Context, raw json.RawMessage) map[string]any {
	var in resolveArgs
	if err := decode(raw, &in); err != nil {
		return toolError(err)
	}
	if in.ThreadID == "" {
		return toolError(fmt.Errorf("thread_id is required"))
	}
	if err := github.AssertResolvable(in.ThreadID); err != nil {
		return toolError(err)
	}
	client, err := github.NewClient(ctx)
	if err != nil {
		return toolError(err)
	}
	if err := client.GraphQL().ResolveThread(ctx, in.ThreadID); err != nil {
		return toolError(fmt.Errorf("resolve thread: %w", err))
	}
	return jsonResult(models.ResolveResult{ThreadID: in.ThreadID, IsResolved: true})
}

func unresolveThread(ctx context.Context, raw json.RawMessage) map[string]any {
	var in resolveArgs
	if err := decode(raw, &in); err != nil {
		return toolError(err)
	}
	if in.ThreadID == "" {
		return toolError(fmt.Errorf("thread_id is required"))
	}
	if err := github.AssertResolvable(in.ThreadID); err != nil {
		return toolError(err)
	}
	client, err := github.NewClient(ctx)
	if err != nil {
		return toolError(err)
	}
	if err := client.GraphQL().UnresolveThread(ctx, in.ThreadID); err != nil {
		return toolError(fmt.Errorf("unresolve thread: %w", err))
	}
	return jsonResult(models.ResolveResult{ThreadID: in.ThreadID, IsResolved: false})
}

func authStatus(ctx context.Context, _ json.RawMessage) map[string]any {
	token := github.ResolveToken()
	if token == "" {
		return toolError(fmt.Errorf("not authenticated: run `pr-agent auth login` or set GITHUB_TOKEN"))
	}
	client, err := github.NewClientWithToken(ctx, token)
	if err != nil {
		return toolError(err)
	}
	login, err := client.AuthenticatedUser(ctx)
	if err != nil {
		return toolError(fmt.Errorf("token invalid: %w", err))
	}
	return jsonResult(map[string]any{"authenticated": true, "user": login})
}

func getAgentGuide(_ context.Context, _ json.RawMessage) map[string]any {
	return textResult(guide.AgentGuide)
}

// --- helpers ---

func clientAndRepo(ctx context.Context, repo string) (*github.Client, string, string, error) {
	client, err := github.NewClient(ctx)
	if err != nil {
		return nil, "", "", err
	}
	owner, name, err := github.ParseRepo(repo)
	if err != nil {
		return nil, "", "", err
	}
	return client, owner, name, nil
}

func decode(raw json.RawMessage, dest any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

func parseKind(s string) (models.CommentKind, error) {
	switch s {
	case "", "inline_review", "inline":
		return models.KindInlineReview, nil
	case "issue_comment", "issue":
		return models.KindIssueComment, nil
	case "review_body", "review":
		return models.KindReviewBody, nil
	default:
		return "", fmt.Errorf("invalid kind %q: use inline_review, issue_comment, or review_body", s)
	}
}

// jsonResult builds a successful MCP tool result with JSON text content.
func jsonResult(v any) map[string]any {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError(fmt.Errorf("encode result: %w", err))
	}
	return textResult(string(b))
}

func textResult(s string) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": s},
		},
	}
}

// toolError builds an MCP tool result flagged as an error.
func toolError(err error) map[string]any {
	return map[string]any{
		"isError": true,
		"content": []map[string]any{
			{"type": "text", "text": err.Error()},
		},
	}
}
