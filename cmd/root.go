package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/lcarva/tektor/cmd/expand"
	"github.com/lcarva/tektor/cmd/validate"
)

var rootCmd = &cobra.Command{
	Use:          "tektor",
	Short:        "Tektor is a validator for Tekton resources.",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(validate.ValidateCmd)
	rootCmd.AddCommand(expand.ExpandCmd)
}
