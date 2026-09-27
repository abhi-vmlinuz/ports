package cli

import (
	"fmt"
	"os"

	"ports/internal/config"
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
	var snapshot bool
	var themeFlag string

	cmd := &cobra.Command{
		Use:   "ports [port]",
		Short: "Find processes listening on local ports",
		Long: `ports is a fast, Linux-first CLI that answers: "What is using this port?"
It directly inspects Linux /proc interfaces without relying on lsof or netstat.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		Example: `  ports                  # Interactive TUI dashboard (default in terminal)
  ports -s               # Static snapshot table
  ports 3000             # Fast inspection card for port 3000
  ports 3000 -w          # Watch port 3000 continuously in TUI
  ports -t catppuccin    # Launch with Catppuccin theme
  ports -t list          # List all 16 available themes
  ports kill 3000        # Safely terminate process on port 3000
  ports --json           # Machine-readable JSON output`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if themeFlag == "list" {
				cfg := config.Load()
				renderer.PrintAvailableThemes(out, cfg.Theme)
				return nil
			}

			if themeFlag != "" && !renderer.IsValidTheme(themeFlag) {
				fmt.Fprintf(os.Stderr, "error: unknown theme '%s'. Run 'ports --theme list' to see available themes.\n", themeFlag)
				os.Exit(2)
			}

			// Explicit watch mode
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
				return renderer.WatchTUI(port, 0, themeFlag)
			}

			// Default behavior when no port is specified:
			// Launch interactive TUI if stdout is a TTY and not requesting snapshot/json.
			if len(args) == 0 && !snapshot && !jsonOutput && renderer.IsTerminal() {
				return renderer.WatchTUI(0, 0, themeFlag)
			}

			theme := renderer.NewTheme(noColor, themeFlag)
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
					if err := jsonRenderer.Render(out, records); err != nil {
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
						fmt.Fprintf(out, "%sPort %d is not in use.%s\n", theme.Dim, port, theme.Reset)
					} else {
						fmt.Fprintf(out, "Port %d is not in use.\n", port)
					}
					os.Exit(1)
				}

				cardRenderer := renderer.NewCardRenderer(theme)
				return cardRenderer.Render(out, records)
			}

			// Listing mode (snapshot table or piped / redirected / non-TTY)
			records, err := discoverer.DiscoverAll()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to discover ports: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				jsonRenderer := renderer.NewJSONRenderer()
				return jsonRenderer.Render(out, records)
			}

			tableRenderer := renderer.NewTableRenderer(theme)
			return tableRenderer.Render(out, records)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output machine-readable JSON")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable color styling")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch ports continuously in an interactive TUI")
	cmd.Flags().BoolVarP(&snapshot, "snapshot", "s", false, "render static snapshot table instead of opening interactive TUI")
	cmd.Flags().BoolVar(&snapshot, "table", false, "alias for --snapshot")
	_ = cmd.Flags().MarkHidden("table")
	cmd.Flags().StringVarP(&themeFlag, "theme", "t", "", "color theme (e.g. catppuccin, nord, cyberpunk, or 'list')")

	cmd.AddCommand(newKillCmd())

	return cmd
}
