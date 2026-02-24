package main

import (
	"context"

	"github.com/tomatopunk/agent-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Stop plugin and remove work dir",
	RunE:  runDelete,
}

var deletePluginID string

func init() {
	deleteCmd.Flags().StringVar(&deletePluginID, "plugin-id", "", "plugin ID (required)")
	_ = deleteCmd.MarkFlagRequired("plugin-id")
}

func runDelete(cmd *cobra.Command, _ []string) error {
	return runtime.New(mustRoot(cmd)).Delete(context.Background(), deletePluginID)
}

func init() { rootCmd.AddCommand(deleteCmd) }
