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
	Short: "Start plugin (re-exec shim keeps plugin as child; block until plugin exits or SIGTERM/SIGINT)",
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
	runExecutable    string
	runArgs          string
	runCPU           string
	runMem           string
	runEnv           string
	runExec          bool // true when we are the re-exec'd shim child (internal)
)

func init() {
	runCmd.Flags().StringVar(&runPluginID, "plugin-id", "", "plugin ID (required)")
	runCmd.Flags().StringVar(&runPluginVersion, "plugin-version", "", "plugin version (injected as PLUGIN_VERSION)")
	runCmd.Flags().StringVar(&runDeviceId, "device-id", "", "device ID (injected as DEVICE_ID)")
	runCmd.Flags().StringVar(&runHostType, "host-type", "", "host type (injected as HOST_TYPE)")
	runCmd.Flags().StringVar(&runHostName, "host-name", "", "host name (injected as HOST_NAME)")
	runCmd.Flags().StringVar(&runBackend, "backend", "binary", "backend: binary | runc")
	runCmd.Flags().StringVar(&runWorkDir, "work-dir", "", "work dir / bundle path (required)")
	runCmd.Flags().StringVar(&runExecutable, "executable", "", "host path to the binary to run (required)")
	runCmd.Flags().StringVar(&runArgs, "args", "", "optional args for the command, comma-separated")
	runCmd.Flags().StringVar(&runCPU, "cpu", "", "cgroup CPU quota")
	runCmd.Flags().StringVar(&runMem, "mem", "", "cgroup memory quota")
	runCmd.Flags().StringVar(&runEnv, "env", "", "env vars, comma-separated KEY=VALUE")
	runCmd.Flags().BoolVar(&runExec, "exec", false, "internal: re-exec'd shim process")
	_ = runCmd.Flags().MarkHidden("exec")
	_ = runCmd.MarkFlagRequired("plugin-id")
	_ = runCmd.MarkFlagRequired("work-dir")
	_ = runCmd.MarkFlagRequired("executable")
}

func runRun(cmd *cobra.Command, _ []string) error {
	root := mustRoot(cmd)

	// Re-exec pattern (like runc): parent only forks child with --exec and blocks; no runtime state in parent.
	if !runExec {
		_, err := os.Executable()
		if err != nil {
			return fmt.Errorf("executable: %w", err)
		}
		argv := []string{"/proc/self/exe", "run", "--exec", "-r", root, "--plugin-id", runPluginID, "--backend", runBackend, "--work-dir", runWorkDir, "--executable", runExecutable}
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
		if runArgs != "" {
			argv = append(argv, "--args", runArgs)
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
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		c.Env = os.Environ()
		return c.Run()
	}

	// We are the shim child: only this process builds runtime state and runs the plugin.
	var env []string
	if runEnv != "" {
		for _, e := range strings.Split(runEnv, ",") {
			env = append(env, strings.TrimSpace(e))
		}
	}
	var args []string
	if runArgs != "" {
		for _, a := range strings.Split(runArgs, ",") {
			args = append(args, strings.TrimSpace(a))
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
		Executable:    runExecutable,
		Args:          args,
		CPU:           runCPU,
		Mem:           runMem,
		Env:           env,
	}
	rt := runtime.New(root)
	return rt.RunAndWait(context.Background(), runBackend, opts)
}

func init() { rootCmd.AddCommand(runCmd) }
