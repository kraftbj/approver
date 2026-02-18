package main

import (
	"fmt"
	"os"

	"github.com/kraft/approver/internal/app"
	"github.com/kraft/approver/internal/config"
	"github.com/kraft/approver/internal/debug"
	"github.com/spf13/cobra"
)

func main() {
	var addRepo string
	var debugMode bool

	rootCmd := &cobra.Command{
		Use:   "approver",
		Short: "A TUI for managing GitHub PR reviews",
		RunE: func(cmd *cobra.Command, args []string) error {
			if debugMode {
				debug.Enable()
				fmt.Fprintln(os.Stderr, "Debug logging to ~/.config/approver/debug.log")
			}
			if addRepo != "" {
				return config.AddRepoSource(addRepo)
			}
			return app.Run()
		},
		SilenceUsage: true,
	}

	rootCmd.Flags().StringVar(&addRepo, "add-repo", "", "register a git repo path and exit")
	rootCmd.Flags().BoolVar(&debugMode, "debug", false, "enable debug logging to ~/.config/approver/debug.log")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
