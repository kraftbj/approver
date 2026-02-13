package main

import (
	"fmt"
	"os"

	"github.com/kraft/approver/internal/app"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "approver",
		Short: "A TUI for managing GitHub PR reviews",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run()
		},
		SilenceUsage: true,
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
