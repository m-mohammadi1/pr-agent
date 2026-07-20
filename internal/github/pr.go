package github

import (
	"context"

	"github.com/m-mohammadi1/pr-agent/internal/models"
)

// FetchPRInfo returns title, description, and basic metadata for a pull request.
func (c *Client) FetchPRInfo(ctx context.Context, owner, repo string, pr int) (*models.PRInfo, error) {
	pull, resp, err := c.rest.PullRequests.Get(ctx, owner, repo, pr)
	if err != nil {
		return nil, apiError("get pull request", resp, err)
	}

	info := &models.PRInfo{
		Repo:   owner + "/" + repo,
		PR:     pr,
		Title:  pull.GetTitle(),
		Body:   pull.GetBody(),
		State:  pull.GetState(),
		Draft:  pull.GetDraft(),
		Merged: pull.GetMerged(),
		URL:    pull.GetHTMLURL(),
	}
	if pull.User != nil {
		info.Author = pull.User.GetLogin()
	}
	if pull.Base != nil {
		info.BaseRef = pull.Base.GetRef()
	}
	if pull.Head != nil {
		info.HeadRef = pull.Head.GetRef()
	}
	if t := pull.GetCreatedAt(); !t.Time.IsZero() {
		info.CreatedAt = t.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	if t := pull.GetUpdatedAt(); !t.Time.IsZero() {
		info.UpdatedAt = t.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	return info, nil
}
