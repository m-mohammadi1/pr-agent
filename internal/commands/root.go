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
		Use:           "pr-agent",
		Short:         "GitHub PR review bridge for agents",
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
	var unresolved bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List PR review threads",
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

			threads, err := client.FetchThreads(ctx, owner, name, flags.PR, unresolved)
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
	cmd.Flags().BoolVar(&unresolved, "unresolved", false, "only return unresolved threads")
	return cmd
}

func newContextCommand() *cobra.Command {
	var flags repoFlags
	var unresolved bool

	cmd := &cobra.Command{
		Use:   "context",
		Short: "Build agent fix-queue with comment and diff context",
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

			threads, err := client.FetchThreads(ctx, owner, name, flags.PR, unresolved)
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
	cmd.Flags().BoolVar(&unresolved, "unresolved", true, "only include unresolved threads")
	return cmd
}

func newReplyCommand() *cobra.Command {
	var repo string
	var pr int
	var commentID string
	var body string

	cmd := &cobra.Command{
		Use:   "reply",
		Short: "Reply to an inline review comment",
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

			created, err := client.CreateReply(ctx, owner, name, pr, id, body)
			if err != nil {
				return wrapAPI(err)
			}

			result := models.ReplyResult{
				CommentID: fmt.Sprintf("%d", created.GetID()),
				Body:      created.GetBody(),
			}
			if created.HTMLURL != nil {
				result.URL = *created.HTMLURL
			}

			return output.WriteJSON(result)
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "repository in owner/name format (required)")
	cmd.Flags().IntVar(&pr, "pr", 0, "pull request number (required)")
	cmd.Flags().StringVar(&commentID, "comment-id", "", "numeric database id of comment to reply to (required)")
	cmd.Flags().StringVar(&body, "body", "", "reply body (required)")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("pr")
	_ = cmd.MarkFlagRequired("comment-id")
	return cmd
}

func newResolveCommand() *cobra.Command {
	var threadID string

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve a review thread",
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
		Use:   "status",
		Short: "Summarize review thread counts for a PR",
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

			threads, err := client.FetchThreads(ctx, owner, name, flags.PR, false)
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
