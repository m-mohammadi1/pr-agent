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

type fetchFlags struct {
	inlineOnly         bool
	noConversation     bool
	includeIssue       bool
	includeReviewBody  bool
	unresolved         bool
}

func (f *fetchFlags) bind(cmd *cobra.Command, defaultUnresolved bool) {
	cmd.Flags().BoolVar(&f.unresolved, "unresolved", defaultUnresolved, "only include unresolved inline threads (conversation comments always included)")
	cmd.Flags().BoolVar(&f.inlineOnly, "inline-only", false, "only fetch inline review threads")
	cmd.Flags().BoolVar(&f.noConversation, "no-conversation", false, "exclude PR conversation and review-body comments")
	cmd.Flags().BoolVar(&f.includeIssue, "include-issue", true, "include top-level PR conversation comments")
	cmd.Flags().BoolVar(&f.includeReviewBody, "include-review-body", true, "include submitted review summary bodies")
}

func (f *fetchFlags) options() github.FetchOptions {
	opts := github.FetchOptions{
		IncludeInline:     true,
		IncludeIssue:      f.includeIssue,
		IncludeReviewBody: f.includeReviewBody,
		UnresolvedOnly:    f.unresolved,
	}
	if f.inlineOnly {
		opts.IncludeIssue = false
		opts.IncludeReviewBody = false
	}
	if f.noConversation {
		opts.IncludeIssue = false
		opts.IncludeReviewBody = false
	}
	return opts
}

// NewRootCommand builds the CLI root.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "pr-agent",
		Short: "GitHub PR review bridge for AI agents",
		Long:  rootLong,
		Example: `  # One-time auth
  pr-agent auth login

  # Get fix queue (primary agent entry point)
  pr-agent context --repo owner/repo --pr 42

  # After fixing locally
  pr-agent reply --repo owner/repo --pr 42 --comment-id 123456 --body "Fixed in abc123"
  pr-agent resolve --thread-id PRRT_kwDO...

  # Verify
  pr-agent status --repo owner/repo --pr 42`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newListCommand())
	root.AddCommand(newContextCommand())
	root.AddCommand(newReplyCommand())
	root.AddCommand(newResolveCommand())
	root.AddCommand(newStatusCommand())
	root.AddCommand(newAuthCommand())

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

func newListCommand() *cobra.Command {
	var flags repoFlags
	var fetch fetchFlags

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List PR review threads and conversation comments",
		Long:    listLong,
		Example: `  pr-agent list --repo owner/repo --pr 42
  pr-agent list --repo owner/repo --pr 42 --unresolved
  pr-agent list --repo owner/repo --pr 42 --inline-only`,
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

			threads, err := client.FetchThreads(ctx, owner, name, flags.PR, fetch.options())
			if err != nil {
				return wrapAPI(fmt.Errorf("list threads: %w", err))
			}

			return output.WriteJSON(models.ListResult{
				Repo:    flags.Repo,
				PR:      flags.PR,
				Threads: threads,
			})
		},
	}

	flags.bind(cmd)
	fetch.bind(cmd, false)
	return cmd
}

func newContextCommand() *cobra.Command {
	var flags repoFlags
	var fetch fetchFlags

	cmd := &cobra.Command{
		Use:     "context",
		Short:   "Build agent fix-queue with comment and diff context",
		Long:    contextLong,
		Example: `  pr-agent context --repo owner/repo --pr 42
  pr-agent context --repo owner/repo --pr 42 --unresolved=false
  pr-agent context --repo owner/repo --pr 42 --no-conversation`,
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

			threads, err := client.FetchThreads(ctx, owner, name, flags.PR, fetch.options())
			if err != nil {
				return wrapAPI(fmt.Errorf("fetch threads: %w", err))
			}

			items := github.BuildContext(threads)
			summary := github.Summarize(threads)

			return output.WriteJSON(models.ContextResult{
				Repo:    flags.Repo,
				PR:      flags.PR,
				Items:   items,
				Summary: summary,
			})
		},
	}

	flags.bind(cmd)
	fetch.bind(cmd, true)
	return cmd
}

func newReplyCommand() *cobra.Command {
	var repo string
	var pr int
	var commentID string
	var body string
	var kind string

	cmd := &cobra.Command{
		Use:     "reply",
		Short:   "Reply to a review or conversation comment",
		Long:    replyLong,
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

func newStatusCommand() *cobra.Command {
	var flags repoFlags

	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Summarize review thread counts for a PR",
		Long:    statusLong,
		Example: `  pr-agent status --repo owner/repo --pr 42`,
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

			threads, err := client.FetchThreads(ctx, owner, name, flags.PR, github.DefaultFetchOptions())
			if err != nil {
				return wrapAPI(fmt.Errorf("fetch threads: %w", err))
			}

			return output.WriteJSON(models.StatusResult{
				Repo:    flags.Repo,
				PR:      flags.PR,
				Summary: github.Summarize(threads),
			})
		},
	}

	flags.bind(cmd)
	return cmd
}
