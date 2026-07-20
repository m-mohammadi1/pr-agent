package github

import (
	"context"

	"github.com/hero/pr-agent/internal/models"
)

// FetchThreads returns review threads for a PR, optionally filtered.
func (c *Client) FetchThreads(ctx context.Context, owner, repo string, pr int, unresolvedOnly bool) ([]models.Thread, error) {
	threads, err := c.gql.FetchReviewThreads(ctx, owner, repo, pr)
	if err != nil {
		return nil, err
	}
	if !unresolvedOnly {
		return threads, nil
	}

	filtered := make([]models.Thread, 0, len(threads))
	for _, t := range threads {
		if !t.IsResolved {
			filtered = append(filtered, t)
		}
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
		})
	}
	return items
}

// Summarize computes aggregate stats for threads.
func Summarize(threads []models.Thread) models.StatusSummary {
	s := models.StatusSummary{Total: len(threads)}
	for _, t := range threads {
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
