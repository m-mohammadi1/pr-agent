package github

import (
	"context"
	"fmt"
	"time"

	gh "github.com/google/go-github/v69/github"
	"github.com/hero/pr-agent/internal/models"
)

// FetchIssueComments returns top-level PR conversation comments.
func (c *Client) FetchIssueComments(ctx context.Context, owner, repo string, pr int) ([]models.Thread, error) {
	opts := &gh.IssueListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	var threads []models.Thread
	for {
		comments, resp, err := c.rest.Issues.ListComments(ctx, owner, repo, pr, opts)
		if err != nil {
			return nil, apiError("list issue comments", resp, err)
		}

		for _, ic := range comments {
			author := ""
			if ic.User != nil {
				author = ic.User.GetLogin()
			}
			threads = append(threads, models.Thread{
				ThreadID:   fmt.Sprintf("IC_%d", ic.GetID()),
				Kind:       models.KindIssueComment,
				IsResolved: false,
				Comments: []models.Comment{{
					ID:        fmt.Sprintf("%d", ic.GetID()),
					Body:      ic.GetBody(),
					Author:    author,
					CreatedAt: ic.GetCreatedAt().Time,
				}},
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return threads, nil
}

// FetchReviewMetas returns all submitted PR reviews (including empty bodies).
func (c *Client) FetchReviewMetas(ctx context.Context, owner, repo string, pr int) ([]models.ReviewMeta, error) {
	opts := &gh.ListOptions{PerPage: 100}

	var out []models.ReviewMeta
	for {
		reviews, resp, err := c.rest.PullRequests.ListReviews(ctx, owner, repo, pr, opts)
		if err != nil {
			return nil, apiError("list reviews", resp, err)
		}

		for _, rv := range reviews {
			author := ""
			if rv.User != nil {
				author = rv.User.GetLogin()
			}
			submittedAt := time.Time{}
			if rv.SubmittedAt != nil {
				submittedAt = rv.SubmittedAt.Time
			}

			out = append(out, models.ReviewMeta{
				ReviewID:    ReviewIDFromDatabaseID(rv.GetID()),
				Author:      author,
				State:       rv.GetState(),
				Body:        rv.GetBody(),
				SubmittedAt: submittedAt,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return out, nil
}

// CreateIssueComment posts a top-level comment on the PR conversation.
func (c *Client) CreateIssueComment(ctx context.Context, owner, repo string, pr int, body string) (*gh.IssueComment, error) {
	comment := &gh.IssueComment{Body: gh.String(body)}
	created, resp, err := c.rest.Issues.CreateComment(ctx, owner, repo, pr, comment)
	if err != nil {
		return nil, apiError("create issue comment", resp, err)
	}
	return created, nil
}
