package main

import (
	"fmt"
	"os"

	"github.com/kraft/approver/internal/app"
	"github.com/kraft/approver/internal/config"
	"github.com/spf13/cobra"
)

func main() {
	var addRepo string

	rootCmd := &cobra.Command{
		Use:   "approver",
		Short: "A TUI for managing GitHub PR reviews",
		RunE: func(cmd *cobra.Command, args []string) error {
			if addRepo != "" {
				return config.AddRepoSource(addRepo)
			}
			return app.Run()
		},
		SilenceUsage: true,
	}

	rootCmd.Flags().StringVar(&addRepo, "add-repo", "", "register a git repo path and exit")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
