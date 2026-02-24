package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tomatopunk/agent-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Query single plugin state",
	RunE:  runState,
}

var statePluginID string
var stateFormat string

func init() {
	stateCmd.Flags().StringVar(&statePluginID, "plugin-id", "", "plugin ID (required)")
	stateCmd.Flags().StringVar(&stateFormat, "format", "text", "output format: text | json")
	_ = stateCmd.MarkFlagRequired("plugin-id")
}

func runState(cmd *cobra.Command, _ []string) error {
	rt := runtime.New(mustRoot(cmd))
	info, err := rt.State(context.Background(), statePluginID)
	if err != nil {
		return err
	}
	if stateFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}
	fmt.Printf("plugin_id: %s\nbackend: %s\nstatus: %s\npid: %d\n", info.PluginID, info.Backend, info.Status, info.Pid)
	return nil
}

func init() { rootCmd.AddCommand(stateCmd) }
