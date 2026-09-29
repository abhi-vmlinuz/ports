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
	var pidOnly bool
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
  ports 3000 -p          # Print only PID for scripts (e.g. kill $(ports 3000 -p))
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
				return fmt.Errorf("unknown theme '%s'. Run 'ports --theme list' to see available themes.", themeFlag)
			}

			discoverer := proc.NewDiscoverer("/proc")

			// Scripting flag: output only PID for port
			if pidOnly {
				if len(args) == 0 {
					return fmt.Errorf("--pid requires a port argument (e.g. ports 3000 --pid)")
				}
				port, err := ParsePortArgument(args[0])
				if err != nil {
					return err
				}
				records, err := discoverer.DiscoverPort(port)
				if err != nil {
					return fmt.Errorf("failed to inspect port %d: %w", port, err)
				}

				seen := make(map[int]bool)
				var pids []int
				for _, r := range records {
					if r.PID > 0 && !seen[r.PID] {
						seen[r.PID] = true
						pids = append(pids, r.PID)
					}
				}

				if len(pids) == 0 {
					return fmt.Errorf("no process found listening on port %d", port)
				}

				for _, pid := range pids {
					fmt.Fprintln(out, pid)
				}
				return nil
			}

			// Explicit watch mode
			if watch {
				var port uint16
				if len(args) == 1 {
					p, err := ParsePortArgument(args[0])
					if err != nil {
						return err
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

			// Single port inspection mode
			if len(args) == 1 {
				port, err := ParsePortArgument(args[0])
				if err != nil {
					return err
				}

				records, err := discoverer.DiscoverPort(port)
				if err != nil {
					return fmt.Errorf("failed to inspect port %d: %w", port, err)
				}

				if jsonOutput {
					if err := renderer.RenderJSON(out, records); err != nil {
						return err
					}
					if len(records) == 0 {
						return fmt.Errorf("port %d is not in use", port)
					}
					return nil
				}

				if len(records) == 0 {
					if theme.Enabled {
						fmt.Fprintf(out, "%sPort %d is not in use.%s\n", theme.Dim, port, theme.Reset)
					} else {
						fmt.Fprintf(out, "Port %d is not in use.\n", port)
					}
					return fmt.Errorf("port %d is not in use", port)
				}

				cardRenderer := renderer.NewCardRenderer(theme)
				return cardRenderer.Render(out, records)
			}

			// Listing mode (snapshot table or piped / redirected / non-TTY)
			records, err := discoverer.DiscoverAll()
			if err != nil {
				return fmt.Errorf("failed to discover ports: %w", err)
			}

			if jsonOutput {
				return renderer.RenderJSON(out, records)
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
	cmd.Flags().BoolVarP(&pidOnly, "pid", "p", false, "output only the PID(s) of the process listening on the specified port")
	cmd.Flags().StringVarP(&themeFlag, "theme", "t", "", "color theme (e.g. catppuccin, nord, cyberpunk, or 'list')")

	cmd.AddCommand(newKillCmd())

	return cmd
}
