package main

import (
	"context"

	"github.com/tomatopunk/agent-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Stop and remove all plugins",
	RunE:  runDestroy,
}

func runDestroy(cmd *cobra.Command, _ []string) error {
	return runtime.New(mustRoot(cmd)).Destroy(context.Background())
}

func init() { rootCmd.AddCommand(destroyCmd) }
