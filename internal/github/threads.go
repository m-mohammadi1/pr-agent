package github

import (
	"context"

	"github.com/hero/pr-agent/internal/models"
)

// FetchOptions controls which comment sources to include.
type FetchOptions struct {
	IncludeInline      bool
	IncludeIssue       bool
	IncludeReviewBody  bool
	UnresolvedOnly     bool
}

// DefaultFetchOptions returns options that include all comment types.
func DefaultFetchOptions() FetchOptions {
	return FetchOptions{
		IncludeInline:     true,
		IncludeIssue:      true,
		IncludeReviewBody: true,
	}
}

// FetchThreads returns review threads and conversation comments for a PR.
func (c *Client) FetchThreads(ctx context.Context, owner, repo string, pr int, opts FetchOptions) ([]models.Thread, error) {
	var all []models.Thread

	if opts.IncludeInline {
		inline, err := c.gql.FetchReviewThreads(ctx, owner, repo, pr)
		if err != nil {
			return nil, err
		}
		all = append(all, inline...)
	}

	if opts.IncludeIssue {
		issue, err := c.FetchIssueComments(ctx, owner, repo, pr)
		if err != nil {
			return nil, err
		}
		all = append(all, issue...)
	}

	if opts.IncludeReviewBody {
		reviews, err := c.FetchReviewBodies(ctx, owner, repo, pr)
		if err != nil {
			return nil, err
		}
		all = append(all, reviews...)
	}

	if !opts.UnresolvedOnly {
		return all, nil
	}

	filtered := make([]models.Thread, 0, len(all))
	for _, t := range all {
		if t.Kind == models.KindInlineReview {
			if !t.IsResolved {
				filtered = append(filtered, t)
			}
			continue
		}
		// Issue and review-body comments have no resolve state; always actionable.
		filtered = append(filtered, t)
	}
	return filtered, nil
}

// BuildContext creates agent-ready fix queue items from threads.
func BuildContext(threads []models.Thread) []models.ContextItem {
	items := make([]models.ContextItem, 0, len(threads))
	for _, t := range threads {
		if len(t.Comments) == 0 {
			continue
		}
		latest := t.Comments[len(t.Comments)-1]

		// Copy comments so JSON output is independent of source slice.
		comments := make([]models.Comment, len(t.Comments))
		copy(comments, t.Comments)

		items = append(items, models.ContextItem{
			ThreadID:   t.ThreadID,
			Kind:       t.Kind,
			File:       t.File,
			Line:       t.Line,
			StartLine:  t.StartLine,
			Side:       t.Side,
			IsResolved: t.IsResolved,
			IsOutdated: t.IsOutdated,
			Body:       latest.Body,
			Author:     latest.Author,
			CommentID:  latest.ID,
			DiffHunk:   latest.DiffHunk,
			Comments:   comments,
		})
	}
	return items
}

// Summarize computes aggregate stats for threads.
func Summarize(threads []models.Thread) models.StatusSummary {
	s := models.StatusSummary{Total: len(threads)}
	for _, t := range threads {
		switch t.Kind {
		case models.KindInlineReview:
			s.Inline++
		case models.KindIssueComment:
			s.Issue++
		case models.KindReviewBody:
			s.ReviewBodies++
		}

		if t.IsResolved {
			s.Resolved++
		} else {
			s.Unresolved++
		}
		if t.IsOutdated {
			s.Outdated++
		}
	}
	return s
}
