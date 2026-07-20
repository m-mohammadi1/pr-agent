package github

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/hero/pr-agent/internal/models"
)

const previewMaxRunes = 160

// ListReviews builds the review-centric index: reviews with item counts + orphans.
func (c *Client) ListReviews(ctx context.Context, owner, repo string, pr int) (*models.ReviewsListResult, error) {
	metas, inline, issues, err := c.fetchReviewSources(ctx, owner, repo, pr)
	if err != nil {
		return nil, err
	}

	byReview := groupInlineByReview(inline)
	// Parent ids that own inline items (used so those threads are not treated as orphans).
	itemParentIDs := make(map[string]struct{}, len(byReview))
	for id := range byReview {
		itemParentIDs[id] = struct{}{}
	}

	reviews := make([]models.ReviewListEntry, 0, len(metas))
	var itemsTotal, itemsUnresolved, itemsResolved int
	for _, meta := range metas {
		items := byReview[meta.ReviewID]
		// Skip empty PullRequestReview shells created by threaded replies
		// (no summary body and no line comments of their own).
		if !isSubstantiveReview(meta.Body, len(items)) {
			continue
		}
		unresolved, resolved := countResolve(items)
		itemsTotal += len(items)
		itemsUnresolved += unresolved
		itemsResolved += resolved

		reviews = append(reviews, models.ReviewListEntry{
			ReviewID:        meta.ReviewID,
			Author:          meta.Author,
			State:           meta.State,
			Preview:         previewText(meta.Body, meta.Author, len(items)),
			SubmittedAt:     meta.SubmittedAt,
			ItemsTotal:      len(items),
			ItemsUnresolved: unresolved,
			ItemsResolved:   resolved,
			CanResolve:      false,
		})
	}

	orphans := make([]models.OrphanListEntry, 0)
	for _, t := range inline {
		if t.ReviewID != "" {
			if _, ok := itemParentIDs[t.ReviewID]; ok {
				continue
			}
		}
		// Orphan: no parent review, or parent review missing from the PR review list.
		entry := orphanFromInline(t)
		orphans = append(orphans, entry)
		itemsTotal++
		if t.IsResolved {
			itemsResolved++
		} else {
			itemsUnresolved++
		}
	}
	for _, t := range issues {
		orphans = append(orphans, orphanFromIssue(t))
	}

	return &models.ReviewsListResult{
		Repo: owner + "/" + repo,
		PR:   pr,
		Summary: models.ReviewsListSummary{
			Reviews:         len(reviews),
			Orphans:         len(orphans),
			ItemsTotal:      itemsTotal,
			ItemsUnresolved: itemsUnresolved,
			ItemsResolved:   itemsResolved,
		},
		Reviews: reviews,
		Orphans: orphans,
	}, nil
}

// GetReview returns the full detail for a review (PRR_) or orphan (IC_ / PRRT_).
func (c *Client) GetReview(ctx context.Context, owner, repo string, pr int, id string) (*models.ReviewDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	metas, inline, issues, err := c.fetchReviewSources(ctx, owner, repo, pr)
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasPrefix(id, "PRR_"):
		return detailForReview(owner, repo, pr, id, metas, inline)
	case strings.HasPrefix(id, "IC_"):
		return detailForIssueOrphan(owner, repo, pr, id, issues)
	case strings.HasPrefix(id, "PRRT_"):
		return detailForInlineOrphan(owner, repo, pr, id, inline)
	default:
		return nil, fmt.Errorf("unknown id %q: expected PRR_, IC_, or PRRT_ prefix", id)
	}
}

// AssertResolvable rejects ids that cannot be resolved on GitHub.
func AssertResolvable(threadID string) error {
	switch {
	case threadID == "":
		return fmt.Errorf("not resolvable: thread_id is required")
	case strings.HasPrefix(threadID, "PRR_"):
		return fmt.Errorf("not resolvable: review summaries cannot be resolved (id %s)", threadID)
	case strings.HasPrefix(threadID, "IC_"):
		return fmt.Errorf("not resolvable: issue comments cannot be resolved (id %s)", threadID)
	case !strings.HasPrefix(threadID, "PRRT_"):
		return fmt.Errorf("not resolvable: expected inline thread id (PRRT_), got %q", threadID)
	default:
		return nil
	}
}

func (c *Client) fetchReviewSources(ctx context.Context, owner, repo string, pr int) (
	metas []models.ReviewMeta,
	inline []models.Thread,
	issues []models.Thread,
	err error,
) {
	metas, err = c.FetchReviewMetas(ctx, owner, repo, pr)
	if err != nil {
		return nil, nil, nil, err
	}
	inline, err = c.gql.FetchReviewThreads(ctx, owner, repo, pr)
	if err != nil {
		return nil, nil, nil, err
	}
	issues, err = c.FetchIssueComments(ctx, owner, repo, pr)
	if err != nil {
		return nil, nil, nil, err
	}
	return metas, inline, issues, nil
}

func groupInlineByReview(inline []models.Thread) map[string][]models.Thread {
	out := make(map[string][]models.Thread)
	for _, t := range inline {
		if t.ReviewID == "" {
			continue
		}
		out[t.ReviewID] = append(out[t.ReviewID], t)
	}
	return out
}

func countResolve(items []models.Thread) (unresolved, resolved int) {
	for _, t := range items {
		if t.IsResolved {
			resolved++
		} else {
			unresolved++
		}
	}
	return unresolved, resolved
}

func detailForReview(owner, repo string, pr int, id string, metas []models.ReviewMeta, inline []models.Thread) (*models.ReviewDetail, error) {
	var meta *models.ReviewMeta
	for i := range metas {
		if metas[i].ReviewID == id {
			meta = &metas[i]
			break
		}
	}
	if meta == nil {
		return nil, fmt.Errorf("review %s not found on PR %d", id, pr)
	}

	items := make([]models.ReviewItem, 0)
	for _, t := range inline {
		if t.ReviewID == id {
			items = append(items, threadToReviewItem(t))
		}
	}

	detail := &models.ReviewDetail{
		Repo:        owner + "/" + repo,
		PR:          pr,
		ID:          meta.ReviewID,
		Kind:        models.KindReviewBody,
		Author:      meta.Author,
		State:       meta.State,
		Body:        meta.Body,
		SubmittedAt: meta.SubmittedAt,
		CanResolve:  false,
		Items:       items,
	}
	detail.Reply = &models.ReplyTarget{
		Kind:      models.KindReviewBody,
		CommentID: strings.TrimPrefix(meta.ReviewID, "PRR_"),
	}
	return detail, nil
}

func detailForIssueOrphan(owner, repo string, pr int, id string, issues []models.Thread) (*models.ReviewDetail, error) {
	for _, t := range issues {
		if t.ThreadID != id {
			continue
		}
		body, author, commentID := latestComment(t)
		return &models.ReviewDetail{
			Repo:       owner + "/" + repo,
			PR:         pr,
			ID:         t.ThreadID,
			Kind:       models.KindIssueComment,
			Author:     author,
			Body:       body,
			CanResolve: false,
			Reply: &models.ReplyTarget{
				Kind:      models.KindIssueComment,
				CommentID: commentID,
			},
			Items: []models.ReviewItem{},
		}, nil
	}
	return nil, fmt.Errorf("orphan issue comment %s not found on PR %d", id, pr)
}

func detailForInlineOrphan(owner, repo string, pr int, id string, inline []models.Thread) (*models.ReviewDetail, error) {
	for _, t := range inline {
		if t.ThreadID != id {
			continue
		}
		item := threadToReviewItem(t)
		body, author, _ := latestComment(t)
		return &models.ReviewDetail{
			Repo:       owner + "/" + repo,
			PR:         pr,
			ID:         t.ThreadID,
			Kind:       models.KindInlineReview,
			Author:     author,
			Body:       body,
			CanResolve: false, // container; item has can_resolve
			Items:      []models.ReviewItem{item},
		}, nil
	}
	return nil, fmt.Errorf("inline thread %s not found on PR %d", id, pr)
}

func threadToReviewItem(t models.Thread) models.ReviewItem {
	body, author, commentID := latestComment(t)
	diffHunk := ""
	if len(t.Comments) > 0 {
		// Prefer first comment's hunk (the line-anchored one); fall back to latest.
		diffHunk = t.Comments[0].DiffHunk
		if diffHunk == "" {
			diffHunk = t.Comments[len(t.Comments)-1].DiffHunk
		}
	}
	comments := make([]models.Comment, len(t.Comments))
	copy(comments, t.Comments)

	return models.ReviewItem{
		ThreadID:       t.ThreadID,
		File:           t.File,
		Line:           t.Line,
		StartLine:      t.StartLine,
		Side:           t.Side,
		IsResolved:     t.IsResolved,
		IsOutdated:     t.IsOutdated,
		CanResolve:     true,
		ReplyCommentID: commentID,
		Body:           body,
		Author:         author,
		DiffHunk:       diffHunk,
		Comments:       comments,
	}
}

func latestComment(t models.Thread) (body, author, commentID string) {
	if len(t.Comments) == 0 {
		return "", "", ""
	}
	c := t.Comments[len(t.Comments)-1]
	return c.Body, c.Author, c.ID
}

func orphanFromInline(t models.Thread) models.OrphanListEntry {
	body, author, _ := latestComment(t)
	unresolved, resolved := 0, 0
	if t.IsResolved {
		resolved = 1
	} else {
		unresolved = 1
	}
	return models.OrphanListEntry{
		ID:              t.ThreadID,
		Kind:            models.KindInlineReview,
		Author:          author,
		Preview:         previewText(body, author, 1),
		File:            t.File,
		Line:            t.Line,
		IsResolved:      t.IsResolved,
		CanResolve:      true,
		ItemsUnresolved: unresolved,
		ItemsResolved:   resolved,
	}
}

func orphanFromIssue(t models.Thread) models.OrphanListEntry {
	body, author, _ := latestComment(t)
	return models.OrphanListEntry{
		ID:              t.ThreadID,
		Kind:            models.KindIssueComment,
		Author:          author,
		Preview:         previewText(body, author, 0),
		CanResolve:      false,
		ItemsUnresolved: 0,
		ItemsResolved:   0,
	}
}

// isSubstantiveReview reports whether a PullRequestReview should appear in the index.
// GitHub creates empty-body reviews when someone replies to an inline thread; those
// are not top-level review summaries and must not be listed as independent reviews.
func isSubstantiveReview(body string, itemCount int) bool {
	return strings.TrimSpace(body) != "" || itemCount > 0
}

func previewText(body, author string, itemCount int) string {
	body = strings.TrimSpace(body)
	if body != "" {
		return truncateRunes(body, previewMaxRunes)
	}
	if itemCount > 0 {
		who := author
		if who == "" {
			who = "reviewer"
		}
		return fmt.Sprintf("(%s · %d line comment(s))", who, itemCount)
	}
	if author != "" {
		return fmt.Sprintf("(%s · empty review body)", author)
	}
	return "(empty)"
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
