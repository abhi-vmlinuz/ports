package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"ports/internal/proc"
	"ports/internal/renderer"

	"github.com/spf13/cobra"
)

func newKillCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "kill <port>",
		Short: "Terminate the process using a port",
		Example: `  ports kill 3000
  ports kill :3000 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := ParsePortArgument(args[0])
			if err != nil {
				cmd.SilenceUsage = true
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(2)
			}

			discoverer := proc.NewDiscoverer("/proc")
			records, err := discoverer.DiscoverPort(port)
			if err != nil {
				cmd.SilenceUsage = true
				fmt.Fprintf(os.Stderr, "error: failed to inspect port %d: %v\n", port, err)
				os.Exit(1)
			}

			if len(records) == 0 {
				cmd.SilenceUsage = true
				fmt.Fprintf(os.Stderr, "error: no process found listening on port %d\n", port)
				os.Exit(1)
			}

			// Check for distinct processes owning this port
			pidMap := make(map[int]string)
			for _, r := range records {
				if r.PID > 0 {
					pidMap[r.PID] = r.Process
				}
			}

			if len(pidMap) == 0 {
				cmd.SilenceUsage = true
				fmt.Fprintf(os.Stderr, "error: port %d is in use, but the process PID could not be determined (permission denied or kernel socket)\n", port)
				fmt.Fprintln(os.Stderr, "hint: try running ports kill with elevated privileges (sudo)")
				os.Exit(1)
			}

			if len(pidMap) > 1 {
				cmd.SilenceUsage = true
				fmt.Fprintf(os.Stderr, "error: multiple processes are associated with port %d; refusing to kill automatically\n\n", port)
				fmt.Fprintf(os.Stderr, "%-8s %s\n", "PID", "PROCESS")
				for pid, name := range pidMap {
					fmt.Fprintf(os.Stderr, "%-8d %s\n", pid, name)
				}
				os.Exit(1)
			}

			// Single PID found
			var targetPID int
			for pid := range pidMap {
				targetPID = pid
				break
			}

			// Target record to display
			var targetRecord = records[0]
			for _, r := range records {
				if r.PID == targetPID {
					targetRecord = r
					break
				}
			}

			theme := renderer.NewTheme(false)

			// Display target information
			fmt.Printf("Port %d is used by:\n\n", port)
			fmt.Printf("  %s\n", targetRecord.Process)
			fmt.Printf("  PID:     %d\n", targetPID)
			if targetRecord.User != nil {
				fmt.Printf("  User:    %s\n", *targetRecord.User)
			}
			if targetRecord.CWD != nil {
				fmt.Printf("  CWD:     %s\n", *targetRecord.CWD)
			}
			if targetRecord.Command != nil {
				fmt.Printf("  Command: %s\n", *targetRecord.Command)
			}
			fmt.Println()

			// Confirmation prompt if not forced
			if !force {
				prompt := fmt.Sprintf("Kill PID %d? [y/N]: ", targetPID)
				if theme.Enabled {
					prompt = fmt.Sprintf("%sKill PID %d?%s %s[y/N]:%s ", theme.Bold, targetPID, theme.Reset, theme.Dim, theme.Reset)
				}
				fmt.Print(prompt)

				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("\nOperation cancelled.")
					return nil
				}

				resp := strings.TrimSpace(strings.ToLower(response))
				if resp != "y" && resp != "yes" {
					fmt.Println("Operation cancelled.")
					return nil
				}
			}

			// Send SIGTERM
			if err := syscall.Kill(targetPID, syscall.SIGTERM); err != nil {
				cmd.SilenceUsage = true
				fmt.Fprintf(os.Stderr, "error: failed to send SIGTERM to PID %d: %v\n", targetPID, err)
				os.Exit(1)
			}

			// Brief check to see if process exited
			time.Sleep(150 * time.Millisecond)
			stillAlive := syscall.Kill(targetPID, 0) == nil

			if theme.Enabled {
				if stillAlive {
					fmt.Printf("%s✓%s sent SIGTERM to PID %d (process still shutting down)\n", theme.Green, theme.Reset, targetPID)
				} else {
					fmt.Printf("%s✓%s sent SIGTERM to PID %d\n", theme.Green, theme.Reset, targetPID)
				}
			} else {
				fmt.Printf("✓ sent SIGTERM to PID %d\n", targetPID)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation for destructive operations")

	return cmd
}
