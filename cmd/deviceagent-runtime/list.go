package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tomatopunk/agent-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plugins and their status",
	RunE:  runList,
}

var listFormat string

func init() {
	listCmd.Flags().StringVar(&listFormat, "format", "text", "output format: text | json")
}

func runList(cmd *cobra.Command, _ []string) error {
	rt := runtime.New(mustRoot(cmd))
	list, err := rt.List(context.Background())
	if err != nil {
		return err
	}
	if listFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}
	for _, i := range list {
		fmt.Printf("%s\t%s\t%s\t%d\n", i.PluginID, i.Backend, i.Status, i.Pid)
	}
	return nil
}

func init() { rootCmd.AddCommand(listCmd) }
