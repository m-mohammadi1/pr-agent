package github

import (
	"testing"
	"time"

	"github.com/hero/pr-agent/internal/models"
)

func TestBuildContextIncludesFullCommentChain(t *testing.T) {
	t1 := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)

	threads := []models.Thread{{
		ThreadID: "PRRT_test",
		Kind:     models.KindInlineReview,
		File:     "foo.go",
		Line:     10,
		Comments: []models.Comment{
			{ID: "1", Body: "first", Author: "bot", CreatedAt: t1},
			{ID: "2", Body: "follow-up", Author: "bot", CreatedAt: t2},
		},
	}}

	items := BuildContext(threads)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.Body != "follow-up" {
		t.Fatalf("expected latest body follow-up, got %q", item.Body)
	}
	if item.CommentID != "2" {
		t.Fatalf("expected latest comment id 2, got %q", item.CommentID)
	}
	if len(item.Comments) != 2 {
		t.Fatalf("expected 2 comments in chain, got %d", len(item.Comments))
	}
}

func TestSummarizeCountsByKind(t *testing.T) {
	threads := []models.Thread{
		{Kind: models.KindInlineReview, IsResolved: false},
		{Kind: models.KindInlineReview, IsResolved: true},
		{Kind: models.KindIssueComment},
		{Kind: models.KindReviewBody},
	}

	s := Summarize(threads)
	if s.Total != 4 || s.Inline != 2 || s.Issue != 1 || s.ReviewBodies != 1 {
		t.Fatalf("unexpected summary: %+v", s)
	}
	if s.Unresolved != 3 || s.Resolved != 1 {
		t.Fatalf("unexpected resolve counts: %+v", s)
	}
}
