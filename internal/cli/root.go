package cli

import (
	"fmt"
	"os"

	"ports/internal/platform"
	"ports/internal/proc"
	"ports/internal/renderer"

	"github.com/spf13/cobra"
)

// Execute runs the root ports command.
func Execute(version string) {
	if err := platform.CheckPlatform(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	rootCmd := NewRootCmd(version)
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints error; exit 2 for usage/flags or 1 for execution
		os.Exit(2)
	}
}

// NewRootCmd constructs the primary Cobra command for ports.
func NewRootCmd(version string) *cobra.Command {
	var jsonOutput bool
	var noColor bool
	var watch bool

	cmd := &cobra.Command{
		Use:   "ports [port]",
		Short: "Find processes listening on local ports",
		Long: `ports is a fast, Linux-first CLI that answers: "What is using this port?"
It directly inspects Linux /proc interfaces without relying on lsof or netstat.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		Example: `  ports
  ports 3000
  ports :3000
  ports --watch
  ports 3000 --watch
  ports kill 3000
  ports --json
  ports 3000 --json`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if watch {
				var port uint16
				if len(args) == 1 {
					p, err := ParsePortArgument(args[0])
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: %v\n", err)
						os.Exit(2)
					}
					port = p
				}
				return renderer.WatchTUI(port, 0)
			}

			theme := renderer.NewTheme(noColor)
			discoverer := proc.NewDiscoverer("/proc")

			// Single port inspection mode
			if len(args) == 1 {
				port, err := ParsePortArgument(args[0])
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(2)
				}

				records, err := discoverer.DiscoverPort(port)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: failed to inspect port %d: %v\n", port, err)
					os.Exit(1)
				}

				if jsonOutput {
					jsonRenderer := renderer.NewJSONRenderer()
					if err := jsonRenderer.Render(os.Stdout, records); err != nil {
						fmt.Fprintf(os.Stderr, "error: %v\n", err)
						os.Exit(1)
					}
					if len(records) == 0 {
						os.Exit(1)
					}
					return nil
				}

				if len(records) == 0 {
					if theme.Enabled {
						fmt.Fprintf(os.Stdout, "%sPort %d is not in use.%s\n", theme.Dim, port, theme.Reset)
					} else {
						fmt.Fprintf(os.Stdout, "Port %d is not in use.\n", port)
					}
					os.Exit(1)
				}

				cardRenderer := renderer.NewCardRenderer(theme)
				return cardRenderer.Render(os.Stdout, records)
			}

			// Listing mode
			records, err := discoverer.DiscoverAll()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to discover ports: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				jsonRenderer := renderer.NewJSONRenderer()
				return jsonRenderer.Render(os.Stdout, records)
			}

			tableRenderer := renderer.NewTableRenderer(theme)
			return tableRenderer.Render(os.Stdout, records)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output machine-readable JSON")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable color styling")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch ports continuously in an interactive TUI")

	cmd.AddCommand(newKillCmd())

	return cmd
}
