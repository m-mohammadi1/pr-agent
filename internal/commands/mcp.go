package commands

import (
	"github.com/m-mohammadi1/pr-agent/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run pr-agent as an MCP stdio server",
		Long:  mcpLong,
		Example: `  # Add to your MCP client config (e.g. ~/.cursor/mcp.json):
  # {
  #   "mcpServers": {
  #     "pr-agent": { "command": "pr-agent", "args": ["mcp"] }
  #   }
  # }
  pr-agent mcp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.Serve(cmd.Context())
		},
	}
}
