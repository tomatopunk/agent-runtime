package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start plugin (daemon monitors and restarts on crash)",
	RunE:  runRun,
}

var (
	runPluginID string
	runBackend  string
	runWorkDir  string
	runConfig   string
	runCPU      string
	runMem      string
	runEnv      string
)

func init() {
	runCmd.Flags().StringVar(&runPluginID, "plugin-id", "", "plugin ID (required)")
	runCmd.Flags().StringVar(&runBackend, "backend", "binary", "backend: binary | runc")
	runCmd.Flags().StringVar(&runWorkDir, "work-dir", "", "work dir / bundle path (required)")
	runCmd.Flags().StringVar(&runConfig, "config", "", "config / executable path (required for binary)")
	runCmd.Flags().StringVar(&runCPU, "cpu", "", "cgroup CPU quota")
	runCmd.Flags().StringVar(&runMem, "mem", "", "cgroup memory quota")
	runCmd.Flags().StringVar(&runEnv, "env", "", "env vars, comma-separated KEY=VALUE")
	_ = runCmd.MarkFlagRequired("plugin-id")
	_ = runCmd.MarkFlagRequired("work-dir")
}

func runRun(cmd *cobra.Command, _ []string) error {
	root := mustRoot(cmd)
	if runBackend == "binary" && runConfig == "" {
		return fmt.Errorf("binary backend requires --config (executable path)")
	}
	var env []string
	if runEnv != "" {
		for _, e := range strings.Split(runEnv, ",") {
			env = append(env, strings.TrimSpace(e))
		}
	}
	rt := runtime.New(root)
	opts := backend.RunOptions{
		PluginID: runPluginID,
		RootDir:  root,
		WorkDir:  runWorkDir,
		Config:   runConfig,
		CPU:      runCPU,
		Mem:      runMem,
		Env:      env,
	}
	return rt.Run(context.Background(), runBackend, opts)
}

func init() { rootCmd.AddCommand(runCmd) }
