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
	// ReviewID links an inline thread to its parent PullRequestReview (PRR_<databaseId>).
	// Empty when the parent review is unknown (orphan inline thread).
	ReviewID string    `json:"review_id,omitempty"`
	Comments []Comment `json:"comments"`
}

// ReviewMeta is a submitted PullRequestReview (summary), including empty-body reviews.
type ReviewMeta struct {
	ReviewID    string    `json:"review_id"`
	Author      string    `json:"author"`
	State       string    `json:"state,omitempty"`
	Body        string    `json:"body"`
	SubmittedAt time.Time `json:"submitted_at,omitempty"`
}

// ReplyTarget tells agents how to reply to a review or orphan.
type ReplyTarget struct {
	Kind      CommentKind `json:"kind"`
	CommentID string      `json:"comment_id"`
}

// ReviewListEntry is a lightweight index row for one submitted review.
type ReviewListEntry struct {
	ReviewID        string    `json:"review_id"`
	Author          string    `json:"author"`
	State           string    `json:"state,omitempty"`
	Preview         string    `json:"preview"`
	SubmittedAt     time.Time `json:"submitted_at,omitempty"`
	ItemsTotal      int       `json:"items_total"`
	ItemsUnresolved int       `json:"items_unresolved"`
	ItemsResolved   int       `json:"items_resolved"`
	CanResolve      bool      `json:"can_resolve"` // always false for reviews
}

// OrphanListEntry is a conversation comment or inline thread not under a listed review.
type OrphanListEntry struct {
	ID              string      `json:"id"`
	Kind            CommentKind `json:"kind"`
	Author          string      `json:"author,omitempty"`
	Preview         string      `json:"preview"`
	File            string      `json:"file,omitempty"`
	Line            int         `json:"line,omitempty"`
	IsResolved      bool        `json:"is_resolved,omitempty"`
	CanResolve      bool        `json:"can_resolve"`
	ItemsUnresolved int         `json:"items_unresolved"`
	ItemsResolved   int         `json:"items_resolved"`
}

// ReviewsListSummary aggregates the review-centric index.
type ReviewsListSummary struct {
	Reviews         int `json:"reviews"`
	Orphans         int `json:"orphans"`
	ItemsTotal      int `json:"items_total"`
	ItemsUnresolved int `json:"items_unresolved"`
	ItemsResolved   int `json:"items_resolved"`
}

// ReviewsListResult is returned by list_reviews / reviews.
type ReviewsListResult struct {
	Repo    string             `json:"repo"`
	PR      int                `json:"pr"`
	Summary ReviewsListSummary `json:"summary"`
	Reviews []ReviewListEntry  `json:"reviews"`
	Orphans []OrphanListEntry  `json:"orphans"`
}

// ReviewItem is a resolvable (or already-resolved) inline thread under a review.
type ReviewItem struct {
	ThreadID       string    `json:"thread_id"`
	File           string    `json:"file,omitempty"`
	Line           int       `json:"line,omitempty"`
	StartLine      int       `json:"start_line,omitempty"`
	Side           string    `json:"side,omitempty"`
	IsResolved     bool      `json:"is_resolved"`
	IsOutdated     bool      `json:"is_outdated"`
	CanResolve     bool      `json:"can_resolve"`
	ReplyCommentID string    `json:"reply_comment_id,omitempty"`
	Body           string    `json:"body,omitempty"`
	Author         string    `json:"author,omitempty"`
	DiffHunk       string    `json:"diff_hunk,omitempty"`
	Comments       []Comment `json:"comments"`
}

// ReviewDetail is the full payload for one review or orphan (get_review).
type ReviewDetail struct {
	Repo        string       `json:"repo"`
	PR          int          `json:"pr"`
	ID          string       `json:"id"`
	Kind        CommentKind  `json:"kind"`
	Author      string       `json:"author,omitempty"`
	State       string       `json:"state,omitempty"`
	Body        string       `json:"body,omitempty"`
	SubmittedAt time.Time    `json:"submitted_at,omitempty"`
	CanResolve  bool         `json:"can_resolve"` // false for review/issue containers
	Reply       *ReplyTarget `json:"reply,omitempty"`
	Items       []ReviewItem `json:"items"`
}

// Comment is a single message within a thread.
type Comment struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	DiffHunk  string    `json:"diff_hunk,omitempty"`
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

// PRInfo is basic pull request metadata (title, description, etc.).
type PRInfo struct {
	Repo      string `json:"repo"`
	PR        int    `json:"pr"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	Draft     bool   `json:"draft"`
	Author    string `json:"author,omitempty"`
	URL       string `json:"url,omitempty"`
	BaseRef   string `json:"base_ref,omitempty"`
	HeadRef   string `json:"head_ref,omitempty"`
	Merged    bool   `json:"merged"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}
