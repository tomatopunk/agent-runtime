package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/runtime"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start plugin (daemon monitors and restarts on crash)",
	RunE:  runRun,
}

var (
	runPluginID      string
	runPluginVersion string
	runDeviceId      string
	runHostType      string
	runHostName      string
	runBackend       string
	runWorkDir       string
	runConfig        string
	runCPU           string
	runMem           string
	runEnv           string
	runNoDaemon      bool
)

func init() {
	runCmd.Flags().StringVar(&runPluginID, "plugin-id", "", "plugin ID (required)")
	runCmd.Flags().StringVar(&runPluginVersion, "plugin-version", "", "plugin version (injected as PLUGIN_VERSION)")
	runCmd.Flags().StringVar(&runDeviceId, "device-id", "", "device ID (injected as DEVICE_ID)")
	runCmd.Flags().StringVar(&runHostType, "host-type", "", "host type (injected as HOST_TYPE)")
	runCmd.Flags().StringVar(&runHostName, "host-name", "", "host name (injected as HOST_NAME)")
	runCmd.Flags().StringVar(&runBackend, "backend", "binary", "backend: binary | runc")
	runCmd.Flags().StringVar(&runWorkDir, "work-dir", "", "work dir / bundle path (required)")
	runCmd.Flags().StringVar(&runConfig, "config", "", "config / executable path (required for binary)")
	runCmd.Flags().StringVar(&runCPU, "cpu", "", "cgroup CPU quota")
	runCmd.Flags().StringVar(&runMem, "mem", "", "cgroup memory quota")
	runCmd.Flags().StringVar(&runEnv, "env", "", "env vars, comma-separated KEY=VALUE")
	runCmd.Flags().BoolVar(&runNoDaemon, "no-daemon", false, "run in foreground (no fork)")
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
	opts := backend.RunOptions{
		PluginID:      runPluginID,
		PluginVersion: runPluginVersion,
		DeviceId:      runDeviceId,
		HostType:      runHostType,
		HostName:      runHostName,
		RootDir:       root,
		WorkDir:       runWorkDir,
		Config:        runConfig,
		CPU:           runCPU,
		Mem:           runMem,
		Env:           env,
	}
	rt := runtime.New(root)

	// Run in this process when user passed --no-daemon (or we're the forked daemon child with --no-daemon).
	if runNoDaemon {
		return rt.Run(context.Background(), runBackend, opts)
	}

	// Default: daemon mode — fork child with --no-daemon so it runs in that process; CLI exits immediately.
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("executable path: %w", err)
	}
	argv := []string{exe, "run", "--no-daemon", "-r", root, "--plugin-id", runPluginID, "--backend", runBackend, "--work-dir", runWorkDir}
	if runPluginVersion != "" {
		argv = append(argv, "--plugin-version", runPluginVersion)
	}
	if runDeviceId != "" {
		argv = append(argv, "--device-id", runDeviceId)
	}
	if runHostType != "" {
		argv = append(argv, "--host-type", runHostType)
	}
	if runHostName != "" {
		argv = append(argv, "--host-name", runHostName)
	}
	if runConfig != "" {
		argv = append(argv, "--config", runConfig)
	}
	if runCPU != "" {
		argv = append(argv, "--cpu", runCPU)
	}
	if runMem != "" {
		argv = append(argv, "--mem", runMem)
	}
	if runEnv != "" {
		argv = append(argv, "--env", runEnv)
	}
	c := exec.Command(argv[0], argv[1:]...)
	c.Env = os.Environ()
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = nil
	if err := c.Start(); err != nil {
		return err
	}
	return nil
}

func init() { rootCmd.AddCommand(runCmd) }
