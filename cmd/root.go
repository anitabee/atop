package cmd

import (
	"fmt"
	"os"
	"time"

	"atop/internal/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var interval time.Duration

var rootCmd = &cobra.Command{
	Use:   "atop",
	Short: "macOS system performance monitor",
	Long: `atop is a real-time system performance monitor for macOS.
Displays CPU, memory, disk I/O, and network metrics without requiring sudo.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(ui.New(interval))
		_, err := p.Run()
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().DurationVarP(&interval, "interval", "i", time.Second, "refresh interval (e.g. 500ms, 2s)")
}
