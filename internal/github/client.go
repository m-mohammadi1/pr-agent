package github

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	gh "github.com/google/go-github/v69/github"
	"github.com/hero/pr-agent/internal/config"
)

// Client wraps REST and GraphQL access to GitHub.
type Client struct {
	rest *gh.Client
	gql  *GraphQL
}

// ResolveToken returns the token from env (GITHUB_TOKEN, GH_TOKEN, TOKEN)
// or the stored global config, in that order. Returns "" if none is found.
func ResolveToken() string {
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN", "TOKEN"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return config.StoredToken()
}

// NewClient creates a GitHub client from the resolved token.
func NewClient(ctx context.Context) (*Client, error) {
	token := ResolveToken()
	if token == "" {
		return nil, fmt.Errorf("no token found: run `pr-agent auth login` (or set GITHUB_TOKEN)")
	}
	return NewClientWithToken(ctx, token)
}

// NewClientWithToken creates a GitHub client from an explicit token.
func NewClientWithToken(ctx context.Context, token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	rest := gh.NewClient(nil).WithAuthToken(token)
	gql, err := NewGraphQL(token)
	if err != nil {
		return nil, err
	}
	_ = ctx
	return &Client{rest: rest, gql: gql}, nil
}

// AuthenticatedUser returns the login of the token owner, validating the token.
func (c *Client) AuthenticatedUser(ctx context.Context) (string, error) {
	user, resp, err := c.rest.Users.Get(ctx, "")
	if err != nil {
		return "", apiError("validate token", resp, err)
	}
	return user.GetLogin(), nil
}

// ParseRepo splits "owner/repo" into owner and name.
func ParseRepo(repo string) (owner, name string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo %q: expected owner/repo", repo)
	}
	return parts[0], parts[1], nil
}

// REST returns the underlying go-github client.
func (c *Client) REST() *gh.Client {
	return c.rest
}

// GraphQL returns the GraphQL helper.
func (c *Client) GraphQL() *GraphQL {
	return c.gql
}

// CreateReply posts a reply to an inline review comment.
func (c *Client) CreateReply(ctx context.Context, owner, repo string, pr int, inReplyTo int64, body string) (*gh.PullRequestComment, error) {
	created, resp, err := c.rest.PullRequests.CreateCommentInReplyTo(ctx, owner, repo, pr, body, inReplyTo)
	if err != nil {
		return nil, apiError("create reply", resp, err)
	}
	return created, nil
}

// ParseCommentID converts a numeric comment ID string to int64.
func ParseCommentID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid comment id %q: must be numeric database id", id)
	}
	return n, nil
}

func apiError(op string, resp *gh.Response, err error) error {
	if resp != nil {
		return fmt.Errorf("%s: %w (status %d)", op, err, resp.StatusCode)
	}
	return fmt.Errorf("%s: %w", op, err)
}
