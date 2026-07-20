package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/hero/pr-agent/internal/github"
	"github.com/hero/pr-agent/internal/models"
	"github.com/hero/pr-agent/internal/output"
	"github.com/spf13/cobra"
)

const (
	exitOK       = 0
	exitUsage    = 1
	exitAPIError = 2
)

// Common flags shared across commands.
type repoFlags struct {
	Repo string
	PR   int
}

func (f *repoFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Repo, "repo", "", "repository in owner/name format (required)")
	cmd.Flags().IntVar(&f.PR, "pr", 0, "pull request number (required)")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("pr")
}

// NewRootCommand builds the CLI root.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "pr-agent",
		Short: "GitHub PR review bridge for humans and AI agents",
		Long:  rootLong,
		Example: `  # One-time auth
  pr-agent auth login

  # How to use (index → detail → act)
  pr-agent info    --repo owner/repo --pr 42
  pr-agent reviews --repo owner/repo --pr 42              # INDEX: which reviews need work?
  pr-agent review  --repo owner/repo --pr 42 --id PRR_123 # DETAIL: load that review fully

  # After fixing locally
  pr-agent reply   --repo owner/repo --pr 42 --comment-id 123456 --body "Fixed in abc123"
  pr-agent resolve --thread-id PRRT_kwDO...

  # Verify
  pr-agent reviews --repo owner/repo --pr 42`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newInfoCommand())
	root.AddCommand(newReviewsCommand())
	root.AddCommand(newReviewCommand())
	root.AddCommand(newReplyCommand())
	root.AddCommand(newResolveCommand())
	root.AddCommand(newUnresolveCommand())
	root.AddCommand(newAuthCommand())
	root.AddCommand(newMCPCommand())

	return root
}

// Execute runs the CLI and exits with appropriate code.
func Execute() {
	root := NewRootCommand()
	if err := root.Execute(); err != nil {
		output.WriteError("%v", err)
		os.Exit(exitCode(err))
	}
}

// ExitCode maps errors to process exit codes.
func ExitCode(err error) int {
	if ue, ok := err.(usageError); ok && ue.usage {
		return exitUsage
	}
	if _, ok := err.(apiError); ok {
		return exitAPIError
	}
	return exitUsage
}

func exitCode(err error) int {
	return ExitCode(err)
}

type usageError struct {
	usage bool
	err   error
}

func (e usageError) Error() string { return e.err.Error() }

type apiError struct{ err error }

func (e apiError) Error() string { return e.err.Error() }

func newClient(ctx context.Context) (*github.Client, error) {
	client, err := github.NewClient(ctx)
	if err != nil {
		return nil, usageError{usage: true, err: err}
	}
	return client, nil
}

func parseRepo(repo string) (owner, name string, err error) {
	owner, name, err = github.ParseRepo(repo)
	if err != nil {
		return "", "", usageError{usage: true, err: err}
	}
	return owner, name, nil
}

func wrapAPI(err error) error {
	if err == nil {
		return nil
	}
	return apiError{err: err}
}

func newReviewsCommand() *cobra.Command {
	var flags repoFlags

	cmd := &cobra.Command{
		Use:     "reviews",
		Short:   "Index: list reviews + orphans (counts only; pick an id next)",
		Long:    reviewsLong,
		Example: `  pr-agent reviews --repo owner/repo --pr 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			owner, name, err := parseRepo(flags.Repo)
			if err != nil {
				return err
			}

			result, err := client.ListReviews(ctx, owner, name, flags.PR)
			if err != nil {
				return wrapAPI(fmt.Errorf("list reviews: %w", err))
			}
			result.Repo = flags.Repo
			return output.WriteJSON(result)
		},
	}

	flags.bind(cmd)
	return cmd
}

func newReviewCommand() *cobra.Command {
	var flags repoFlags
	var id string

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Detail: load one review/orphan by --id (full items + comments)",
		Long:  reviewLong,
		Example: `  pr-agent review --repo owner/repo --pr 42 --id PRR_123
  pr-agent review --repo owner/repo --pr 42 --id IC_999
  pr-agent review --repo owner/repo --pr 42 --id PRRT_kwDO...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			owner, name, err := parseRepo(flags.Repo)
			if err != nil {
				return err
			}
			if id == "" {
				return usageError{usage: true, err: fmt.Errorf("--id is required")}
			}

			result, err := client.GetReview(ctx, owner, name, flags.PR, id)
			if err != nil {
				return wrapAPI(fmt.Errorf("get review: %w", err))
			}
			result.Repo = flags.Repo
			return output.WriteJSON(result)
		},
	}

	flags.bind(cmd)
	cmd.Flags().StringVar(&id, "id", "", "review or orphan id: PRR_..., IC_..., or PRRT_... (required)")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newInfoCommand() *cobra.Command {
	var flags repoFlags

	cmd := &cobra.Command{
		Use:     "info",
		Short:   "Get PR title, description, and basic metadata",
		Long:    infoLong,
		Example: `  pr-agent info --repo owner/repo --pr 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			owner, name, err := parseRepo(flags.Repo)
			if err != nil {
				return err
			}

			result, err := client.FetchPRInfo(ctx, owner, name, flags.PR)
			if err != nil {
				return wrapAPI(fmt.Errorf("get pr info: %w", err))
			}
			result.Repo = flags.Repo
			return output.WriteJSON(result)
		},
	}

	flags.bind(cmd)
	return cmd
}

func newReplyCommand() *cobra.Command {
	var repo string
	var pr int
	var commentID string
	var body string
	var kind string

	cmd := &cobra.Command{
		Use:   "reply",
		Short: "Reply to a review or conversation comment",
		Long:  replyLong,
		Example: `  pr-agent reply --repo owner/repo --pr 42 --comment-id 123456 --body "Fixed in abc123"
  pr-agent reply --repo owner/repo --pr 42 --comment-id 789 --kind issue_comment --body "Addressed"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			owner, name, err := parseRepo(repo)
			if err != nil {
				return err
			}

			if body == "" {
				return usageError{usage: true, err: fmt.Errorf("--body is required")}
			}

			id, err := github.ParseCommentID(commentID)
			if err != nil {
				return usageError{usage: true, err: err}
			}

			commentKind, err := parseCommentKind(kind)
			if err != nil {
				return usageError{usage: true, err: err}
			}

			result, err := client.Reply(ctx, owner, name, pr, commentKind, id, body)
			if err != nil {
				return wrapAPI(err)
			}

			return output.WriteJSON(result)
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "repository in owner/name format (required)")
	cmd.Flags().IntVar(&pr, "pr", 0, "pull request number (required)")
	cmd.Flags().StringVar(&commentID, "comment-id", "", "numeric database id of comment to reply to (required)")
	cmd.Flags().StringVar(&body, "body", "", "reply body (required)")
	cmd.Flags().StringVar(&kind, "kind", "inline_review", "comment kind: inline_review, issue_comment, review_body")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("pr")
	_ = cmd.MarkFlagRequired("comment-id")
	return cmd
}

func parseCommentKind(s string) (models.CommentKind, error) {
	switch s {
	case "inline_review", "inline":
		return models.KindInlineReview, nil
	case "issue_comment", "issue":
		return models.KindIssueComment, nil
	case "review_body", "review":
		return models.KindReviewBody, nil
	default:
		return "", fmt.Errorf("invalid --kind %q: use inline_review, issue_comment, or review_body", s)
	}
}

func newResolveCommand() *cobra.Command {
	var threadID string

	cmd := &cobra.Command{
		Use:     "resolve",
		Short:   "Resolve an inline review thread",
		Long:    resolveLong,
		Example: `  pr-agent resolve --thread-id PRRT_kwDOIy9Ugs6SLd_L`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			if threadID == "" {
				return usageError{usage: true, err: fmt.Errorf("--thread-id is required")}
			}
			if err := github.AssertResolvable(threadID); err != nil {
				return usageError{usage: true, err: err}
			}

			if err := client.GraphQL().ResolveThread(ctx, threadID); err != nil {
				return wrapAPI(fmt.Errorf("resolve thread: %w", err))
			}

			return output.WriteJSON(models.ResolveResult{
				ThreadID:   threadID,
				IsResolved: true,
			})
		},
	}

	cmd.Flags().StringVar(&threadID, "thread-id", "", "GraphQL thread id (required)")
	_ = cmd.MarkFlagRequired("thread-id")
	return cmd
}

func newUnresolveCommand() *cobra.Command {
	var threadID string

	cmd := &cobra.Command{
		Use:     "unresolve",
		Short:   "Unresolve an inline review thread",
		Long:    unresolveLong,
		Example: `  pr-agent unresolve --thread-id PRRT_kwDOIy9Ugs6SLd_L`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			if threadID == "" {
				return usageError{usage: true, err: fmt.Errorf("--thread-id is required")}
			}
			if err := github.AssertResolvable(threadID); err != nil {
				return usageError{usage: true, err: err}
			}

			if err := client.GraphQL().UnresolveThread(ctx, threadID); err != nil {
				return wrapAPI(fmt.Errorf("unresolve thread: %w", err))
			}

			return output.WriteJSON(models.ResolveResult{
				ThreadID:   threadID,
				IsResolved: false,
			})
		},
	}

	cmd.Flags().StringVar(&threadID, "thread-id", "", "GraphQL thread id (required)")
	_ = cmd.MarkFlagRequired("thread-id")
	return cmd
}
