package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hero/pr-agent/internal/models"
)

const graphQLEndpoint = "https://api.github.com/graphql"

// GraphQL provides GitHub GraphQL operations.
type GraphQL struct {
	token  string
	client *http.Client
}

// NewGraphQL creates a GraphQL client.
func NewGraphQL(token string) (*GraphQL, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	return &GraphQL{
		token:  token,
		client: http.DefaultClient,
	}, nil
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   map[string]any `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type reviewThreadsQuery struct {
	Repository struct {
		PullRequest struct {
			ReviewThreads struct {
				Nodes    []reviewThreadNode `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"reviewThreads"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type reviewThreadNode struct {
	ID           string `json:"id"`
	IsResolved   bool   `json:"isResolved"`
	IsOutdated   bool   `json:"isOutdated"`
	Path         string `json:"path"`
	Line         *int   `json:"line"`
	StartLine    *int   `json:"startLine"`
	OriginalLine *int   `json:"originalLine"`
	DiffSide     string `json:"diffSide"`
	Comments     struct {
		Nodes    []reviewCommentNode `json:"nodes"`
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
	} `json:"comments"`
}

type reviewCommentNode struct {
	ID         string `json:"id"`
	DatabaseID int64  `json:"databaseId"`
	Body       string `json:"body"`
	DiffHunk   string `json:"diffHunk"`
	CreatedAt  string `json:"createdAt"`
	Author     *struct {
		Login string `json:"login"`
	} `json:"author"`
}

type threadCommentsQuery struct {
	Node struct {
		Comments struct {
			Nodes    []reviewCommentNode `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"comments"`
	} `json:"node"`
}

// FetchReviewThreads returns all inline review threads for a PR with full comment history.
func (g *GraphQL) FetchReviewThreads(ctx context.Context, owner, repo string, pr int) ([]models.Thread, error) {
	const query = `
query($owner: String!, $repo: String!, $pr: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100, after: $cursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          originalLine
          diffSide
          comments(first: 50) {
            pageInfo { hasNextPage endCursor }
            nodes {
              id
              databaseId
              body
              diffHunk
              createdAt
              author { login }
            }
          }
        }
      }
    }
  }
}`

	var all []reviewThreadNode
	cursor := ""

	for {
		vars := map[string]any{
			"owner": owner,
			"repo":  repo,
			"pr":    pr,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		var data reviewThreadsQuery
		if err := g.do(ctx, query, vars, &data); err != nil {
			return nil, err
		}

		threads := data.Repository.PullRequest.ReviewThreads
		for i := range threads.Nodes {
			node := &threads.Nodes[i]
			if node.Comments.PageInfo.HasNextPage {
				more, err := g.fetchThreadComments(ctx, node.ID, node.Comments.PageInfo.EndCursor)
				if err != nil {
					return nil, err
				}
				node.Comments.Nodes = append(node.Comments.Nodes, more...)
			}
		}
		all = append(all, threads.Nodes...)

		if !threads.PageInfo.HasNextPage {
			break
		}
		cursor = threads.PageInfo.EndCursor
	}

	return mapInlineThreads(all), nil
}

func (g *GraphQL) fetchThreadComments(ctx context.Context, threadID, cursor string) ([]reviewCommentNode, error) {
	const query = `
query($threadId: ID!, $cursor: String) {
  node(id: $threadId) {
    ... on PullRequestReviewThread {
      comments(first: 50, after: $cursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          databaseId
          body
          diffHunk
          createdAt
          author { login }
        }
      }
    }
  }
}`

	var all []reviewCommentNode
	for {
		vars := map[string]any{"threadId": threadID}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		var data threadCommentsQuery
		if err := g.do(ctx, query, vars, &data); err != nil {
			return nil, err
		}

		comments := data.Node.Comments
		all = append(all, comments.Nodes...)

		if !comments.PageInfo.HasNextPage {
			break
		}
		cursor = comments.PageInfo.EndCursor
	}
	return all, nil
}

// ResolveThread marks a review thread as resolved. Idempotent.
func (g *GraphQL) ResolveThread(ctx context.Context, threadID string) error {
	const mutation = `
mutation($threadId: ID!) {
  resolveReviewThread(input: {threadId: $threadId}) {
    thread { isResolved }
  }
}`

	var data struct {
		ResolveReviewThread struct {
			Thread struct {
				IsResolved bool `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	}

	if err := g.do(ctx, mutation, map[string]any{"threadId": threadID}, &data); err != nil {
		return err
	}
	return nil
}

// UnresolveThread marks a review thread as unresolved. Idempotent.
func (g *GraphQL) UnresolveThread(ctx context.Context, threadID string) error {
	const mutation = `
mutation($threadId: ID!) {
  unresolveReviewThread(input: {threadId: $threadId}) {
    thread { isResolved }
  }
}`

	var data struct {
		UnresolveReviewThread struct {
			Thread struct {
				IsResolved bool `json:"isResolved"`
			} `json:"thread"`
		} `json:"unresolveReviewThread"`
	}

	if err := g.do(ctx, mutation, map[string]any{"threadId": threadID}, &data); err != nil {
		return err
	}
	return nil
}

func mapInlineThreads(nodes []reviewThreadNode) []models.Thread {
	out := make([]models.Thread, 0, len(nodes))
	for _, n := range nodes {
		thread := models.Thread{
			ThreadID:   n.ID,
			Kind:       models.KindInlineReview,
			File:       n.Path,
			Side:       n.DiffSide,
			IsResolved: n.IsResolved,
			IsOutdated: n.IsOutdated,
		}
		if n.Line != nil {
			thread.Line = *n.Line
		}
		if n.StartLine != nil {
			thread.StartLine = *n.StartLine
		}

		for _, c := range n.Comments.Nodes {
			thread.Comments = append(thread.Comments, mapReviewComment(c))
		}
		out = append(out, thread)
	}
	return out
}

func mapReviewComment(c reviewCommentNode) models.Comment {
	author := ""
	if c.Author != nil {
		author = c.Author.Login
	}
	createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
	return models.Comment{
		ID:        fmt.Sprintf("%d", c.DatabaseID),
		Body:      c.Body,
		Author:    author,
		CreatedAt: createdAt,
		DiffHunk:  c.DiffHunk,
	}
}

func (g *GraphQL) do(ctx context.Context, query string, variables map[string]any, dest any) error {
	reqBody := graphQLRequest{Query: query, Variables: variables}

	var resp graphQLResponse
	if err := postJSON(ctx, g.client, graphQLEndpoint, g.token, reqBody, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("graphql: %s", resp.Errors[0].Message)
	}

	raw, err := marshalJSON(resp.Data)
	if err != nil {
		return err
	}
	return unmarshalJSON(raw, dest)
}
