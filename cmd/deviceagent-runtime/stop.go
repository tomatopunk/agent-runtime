package main

import (
	"context"

	"github.com/tomatopunk/agent-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop plugin",
	RunE:  runStop,
}

var stopPluginID string

func init() {
	stopCmd.Flags().StringVar(&stopPluginID, "plugin-id", "", "plugin ID (required)")
	_ = stopCmd.MarkFlagRequired("plugin-id")
}

func runStop(cmd *cobra.Command, _ []string) error {
	return runtime.New(mustRoot(cmd)).Stop(context.Background(), stopPluginID)
}

func init() { rootCmd.AddCommand(stopCmd) }
