package main

import (
	"context"
	"io"
	"os"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Read plugin log",
	RunE:  runLog,
}

var logPluginID string
var logFormat string
var logLength int

func init() {
	logCmd.Flags().StringVar(&logPluginID, "plugin-id", "", "plugin ID (required)")
	logCmd.Flags().StringVar(&logFormat, "format", "text", "output format: text | json")
	logCmd.Flags().IntVar(&logLength, "length", 0, "max lines (0=all)")
	_ = logCmd.MarkFlagRequired("plugin-id")
}

func runLog(cmd *cobra.Command, _ []string) error {
	rt := runtime.New(mustRoot(cmd))
	r, err := rt.Log(context.Background(), logPluginID, backend.LogOptions{Format: logFormat, Length: logLength})
	if err != nil {
		return err
	}
	if c, ok := r.(io.Closer); ok {
		defer c.Close()
	}
	_, err = io.Copy(os.Stdout, r)
	return err
}

func init() { rootCmd.AddCommand(logCmd) }
