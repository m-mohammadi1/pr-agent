package github

import (
	"strings"
	"testing"
	"time"

	"github.com/hero/pr-agent/internal/models"
)

func TestAssertResolvable(t *testing.T) {
	if err := AssertResolvable("PRRT_abc"); err != nil {
		t.Fatalf("expected PRRT_ resolvable, got %v", err)
	}
	for _, id := range []string{"PRR_1", "IC_2", "xyz", ""} {
		if err := AssertResolvable(id); err == nil {
			t.Fatalf("expected %q not resolvable", id)
		}
	}
}

func TestThreadToReviewItem(t *testing.T) {
	t1 := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	thread := models.Thread{
		ThreadID:   "PRRT_1",
		Kind:       models.KindInlineReview,
		File:       "a.go",
		Line:       10,
		IsResolved: false,
		ReviewID:   "PRR_9",
		Comments: []models.Comment{
			{ID: "1", Body: "first", Author: "bot", CreatedAt: t1, DiffHunk: "@@ -1 +1 @@"},
			{ID: "2", Body: "second", Author: "bot", CreatedAt: t2},
		},
	}
	item := threadToReviewItem(thread)
	if !item.CanResolve {
		t.Fatal("expected can_resolve true")
	}
	if item.ReplyCommentID != "2" {
		t.Fatalf("reply id: got %q", item.ReplyCommentID)
	}
	if item.Body != "second" {
		t.Fatalf("body: got %q", item.Body)
	}
	if item.DiffHunk != "@@ -1 +1 @@" {
		t.Fatalf("diff hunk: got %q", item.DiffHunk)
	}
	if len(item.Comments) != 2 {
		t.Fatalf("comments: got %d", len(item.Comments))
	}
}

func TestPreviewText(t *testing.T) {
	if got := previewText("hello world", "a", 0); got != "hello world" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("x", 200)
	got := previewText(long, "a", 0)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected truncation, got len %d", len(got))
	}
	if got := previewText("", "jasper", 3); got != "(jasper · 3 line comment(s))" {
		t.Fatalf("empty body preview: %q", got)
	}
}

func TestGroupInlineByReview(t *testing.T) {
	inline := []models.Thread{
		{ThreadID: "PRRT_1", ReviewID: "PRR_1", IsResolved: false},
		{ThreadID: "PRRT_2", ReviewID: "PRR_1", IsResolved: true},
		{ThreadID: "PRRT_3", ReviewID: "", IsResolved: false},
	}
	by := groupInlineByReview(inline)
	if len(by["PRR_1"]) != 2 {
		t.Fatalf("expected 2 under PRR_1, got %d", len(by["PRR_1"]))
	}
	u, r := countResolve(by["PRR_1"])
	if u != 1 || r != 1 {
		t.Fatalf("counts unresolved=%d resolved=%d", u, r)
	}
}

func TestIsSubstantiveReview(t *testing.T) {
	if isSubstantiveReview("", 0) {
		t.Fatal("empty reply shell should be skipped")
	}
	if !isSubstantiveReview("## Jasper review", 0) {
		t.Fatal("summary-only review should be kept")
	}
	if !isSubstantiveReview("", 3) {
		t.Fatal("line-comment review with empty body should be kept")
	}
	if !isSubstantiveReview("notes", 2) {
		t.Fatal("body + items should be kept")
	}
}

func TestOrphanFromIssue(t *testing.T) {
	t1 := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	entry := orphanFromIssue(models.Thread{
		ThreadID: "IC_99",
		Kind:     models.KindIssueComment,
		Comments: []models.Comment{{ID: "99", Body: "Dependency Security Scan: 2 findings", Author: "bot", CreatedAt: t1}},
	})
	if entry.CanResolve {
		t.Fatal("issue orphan must not be resolvable")
	}
	if entry.Kind != models.KindIssueComment {
		t.Fatalf("kind: %s", entry.Kind)
	}
	if !strings.Contains(entry.Preview, "Dependency Security Scan") {
		t.Fatalf("preview: %q", entry.Preview)
	}
}
