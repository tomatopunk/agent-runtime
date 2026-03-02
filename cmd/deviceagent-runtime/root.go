package main

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "deviceagent-runtime",
	Short: "Unified runtime CLI with binary and runc backends",
	Long: `Agent invokes this binary only; it does not call runc directly.
This runtime provides unified logs and list/state semantics; daemon/restart is implemented by the caller.`,
}

func init() {
	rootCmd.PersistentFlags().StringP("root", "r", "", "runtime root dir (required)")
}

// mustRoot returns --root from the root command's PersistentFlags; exits if unset.
func mustRoot(cmd *cobra.Command) string {
	root, err := cmd.Root().PersistentFlags().GetString("root")
	if err != nil || root == "" {
		zap.L().Error("must specify --root")
		os.Exit(2)
	}
	return root
}
