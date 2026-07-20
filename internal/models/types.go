package models

import "time"

// CommentKind identifies where a review comment originated.
type CommentKind string

const (
	KindInlineReview CommentKind = "inline_review"
	KindIssueComment CommentKind = "issue_comment"
	KindReviewBody   CommentKind = "review_body"
)

// Thread represents a normalized review thread or top-level PR comment.
type Thread struct {
	ThreadID   string      `json:"thread_id"`
	Kind       CommentKind `json:"kind"`
	File       string      `json:"file,omitempty"`
	Line       int         `json:"line,omitempty"`
	StartLine  int         `json:"start_line,omitempty"`
	Side       string      `json:"side,omitempty"`
	IsResolved bool        `json:"is_resolved"`
	IsOutdated bool        `json:"is_outdated"`
	Comments   []Comment   `json:"comments"`
}

// Comment is a single message within a thread.
type Comment struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	DiffHunk  string    `json:"diff_hunk,omitempty"`
}

// ListResult is returned by the list command.
type ListResult struct {
	Repo    string   `json:"repo"`
	PR      int      `json:"pr"`
	Threads []Thread `json:"threads"`
}

// ContextResult is returned by the context command.
type ContextResult struct {
	Repo    string        `json:"repo"`
	PR      int           `json:"pr"`
	Items   []ContextItem `json:"items"`
	Summary StatusSummary `json:"summary"`
}

// ContextItem is a fix-queue entry with diff context for agents.
type ContextItem struct {
	ThreadID   string      `json:"thread_id"`
	Kind       CommentKind `json:"kind"`
	File       string      `json:"file,omitempty"`
	Line       int         `json:"line,omitempty"`
	StartLine  int         `json:"start_line,omitempty"`
	Side       string      `json:"side,omitempty"`
	IsResolved bool        `json:"is_resolved"`
	IsOutdated bool        `json:"is_outdated"`
	// Latest comment (convenience for agents).
	Body      string `json:"body"`
	Author    string `json:"author"`
	CommentID string `json:"comment_id"`
	DiffHunk  string `json:"diff_hunk,omitempty"`
	// Full conversation in chronological order.
	Comments []Comment `json:"comments"`
}

// StatusSummary holds aggregate PR review comment stats.
type StatusSummary struct {
	Total        int `json:"total"`
	Unresolved   int `json:"unresolved"`
	Resolved     int `json:"resolved"`
	Outdated     int `json:"outdated"`
	Inline       int `json:"inline"`
	Issue        int `json:"issue"`
	ReviewBodies int `json:"review_bodies"`
}

// StatusResult is returned by the status command.
type StatusResult struct {
	Repo    string        `json:"repo"`
	PR      int           `json:"pr"`
	Summary StatusSummary `json:"summary"`
}

// ReplyResult is returned by the reply command.
type ReplyResult struct {
	CommentID string `json:"comment_id"`
	Body      string `json:"body"`
	URL       string `json:"url,omitempty"`
}

// ResolveResult is returned by the resolve command.
type ResolveResult struct {
	ThreadID   string `json:"thread_id"`
	IsResolved bool   `json:"is_resolved"`
}
