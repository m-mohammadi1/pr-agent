package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/m-mohammadi1/pr-agent/internal/config"
	"github.com/m-mohammadi1/pr-agent/internal/github"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate pr-agent (one-time setup)",
		Long:  authLong,
		Example: `  pr-agent auth login
  pr-agent auth login --from-gh
  pr-agent auth status
  pr-agent auth logout`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd, loginOptions{})
		},
	}

	root.AddCommand(newAuthLoginCommand())
	root.AddCommand(newAuthStatusCommand())
	root.AddCommand(newAuthLogoutCommand())
	return root
}

type loginOptions struct {
	token     string
	fromGH    bool
	assumeYes bool
}

func newAuthLoginCommand() *cobra.Command {
	var opts loginOptions

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate and store your GitHub token",
		Long:  authLoginLong,
		Example: `  pr-agent auth login
  pr-agent auth login --from-gh
  pr-agent auth login --token ghp_xxx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.token, "token", "", "provide a token non-interactively")
	cmd.Flags().BoolVar(&opts.fromGH, "from-gh", false, "use the token from `gh auth token` without prompting")
	cmd.Flags().BoolVarP(&opts.assumeYes, "yes", "y", false, "assume yes for prompts")
	return cmd
}

func runLogin(cmd *cobra.Command, opts loginOptions) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	token, source, err := acquireToken(opts)
	if err != nil {
		return err
	}

	client, err := github.NewClientWithToken(ctx, token)
	if err != nil {
		return usageError{usage: true, err: err}
	}

	login, err := client.AuthenticatedUser(ctx)
	if err != nil {
		return usageError{usage: true, err: fmt.Errorf("token is invalid or lacks access: %w", err)}
	}

	path, err := config.Save(&config.Config{Token: token, User: login})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Authenticated as %s (via %s)\n", login, source)
	fmt.Fprintf(os.Stderr, "Token saved to %s\n", path)
	fmt.Fprintln(os.Stderr, "You can now run pr-agent from any directory without setting a token.")
	return nil
}

// acquireToken resolves a token from flags or interactive prompts.
func acquireToken(opts loginOptions) (token, source string, err error) {
	if opts.token != "" {
		return strings.TrimSpace(opts.token), "flag", nil
	}
	if opts.fromGH {
		t, err := ghToken()
		if err != nil {
			return "", "", err
		}
		return t, "gh", nil
	}

	// Interactive flow.
	ghAvailable := false
	var ghTok string
	if t, err := ghToken(); err == nil && t != "" {
		ghAvailable = true
		ghTok = t
	}

	if ghAvailable {
		fmt.Fprintf(os.Stderr, "Found a GitHub CLI token (%s).\n", redactToken(ghTok))
		if opts.assumeYes || promptYesNo("Use this token?", true) {
			return ghTok, "gh", nil
		}
	} else {
		fmt.Fprintln(os.Stderr, "GitHub CLI token not found.")
	}

	fmt.Fprintln(os.Stderr, "Create a token at https://github.com/settings/tokens (fine-grained: Pull requests Read & write, Contents Read).")
	t, err := promptSecret("Paste your GitHub token: ")
	if err != nil {
		return "", "", err
	}
	t = strings.TrimSpace(t)
	if t == "" {
		return "", "", usageError{usage: true, err: fmt.Errorf("no token provided")}
	}
	return t, "manual", nil
}

func ghToken() (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("`gh` not found: install GitHub CLI or paste a token manually")
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("`gh auth token` failed: run `gh auth login` first")
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("`gh auth token` returned an empty token")
	}
	return token, nil
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long:  authStatusLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			source := tokenSource()
			token := github.ResolveToken()
			if token == "" {
				fmt.Fprintln(os.Stderr, "Not authenticated. Run `pr-agent auth login`.")
				return usageError{usage: true, err: fmt.Errorf("not authenticated")}
			}

			client, err := github.NewClientWithToken(ctx, token)
			if err != nil {
				return usageError{usage: true, err: err}
			}
			login, err := client.AuthenticatedUser(ctx)
			if err != nil {
				return usageError{usage: true, err: fmt.Errorf("stored token is invalid: %w", err)}
			}

			fmt.Fprintf(os.Stderr, "Authenticated as %s\n", login)
			fmt.Fprintf(os.Stderr, "Token source: %s\n", source)
			fmt.Fprintf(os.Stderr, "Token: %s\n", redactToken(token))
			return nil
		},
	}
}

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored token",
		Long:  authLogoutLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, existed, err := config.Delete()
			if err != nil {
				return err
			}
			if !existed {
				fmt.Fprintln(os.Stderr, "No stored token to remove.")
				return nil
			}
			fmt.Fprintf(os.Stderr, "Removed stored token (%s)\n", path)
			return nil
		},
	}
}

// tokenSource reports where the active token comes from.
func tokenSource() string {
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN", "TOKEN"} {
		if os.Getenv(k) != "" {
			return "env:" + k
		}
	}
	if config.StoredToken() != "" {
		if p, err := config.Path(); err == nil {
			return "config:" + p
		}
		return "config"
	}
	return "none"
}

func promptYesNo(question string, def bool) bool {
	suffix := "[Y/n]"
	if !def {
		suffix = "[y/N]"
	}
	fmt.Fprintf(os.Stderr, "%s %s ", question, suffix)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return def
	}
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

func promptSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}
		return string(b), nil
	}
	// Fallback for non-TTY (e.g. piped input).
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("read token: %w", err)
	}
	return line, nil
}

func redactToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:2] + "****" + token[len(token)-4:]
}
