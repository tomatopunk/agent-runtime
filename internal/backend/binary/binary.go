package binary

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/state"
)

// Backend runs processes on the host; optional cgroup, log to file.
type Backend struct {
	state *state.Manager
	mu    sync.Mutex
	// pluginID -> cmd for processes started by this process (used by Stop to signal)
	running map[string]*exec.Cmd
}

func New(stateManager *state.Manager) *Backend {
	return &Backend{state: stateManager, running: make(map[string]*exec.Cmd)}
}

func (b *Backend) Run(ctx context.Context, opts backend.RunOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if opts.PluginID == "" || opts.WorkDir == "" || opts.Config == "" {
		return fmt.Errorf("plugin_id, work_dir and config are required")
	}

	cmd := exec.CommandContext(ctx, opts.Config)
	cmd.Dir = opts.WorkDir
	// Inherit host env, inject runtime env (binary has no fs isolation so HOST_DIR=/), then add opts.Env
	cmd.Env = make([]string, 0, len(os.Environ())+6+len(opts.Env))
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env,
		"PLUGIN_ID="+opts.PluginID,
		"PLUGIN_VERSION="+opts.PluginVersion,
		"DEVICE_ID="+opts.DeviceId,
		"HOST_TYPE="+opts.HostType,
		"HOST_NAME="+opts.HostName,
		"HOST_DIR=/", // binary does not isolate fs
	)
	cmd.Env = append(cmd.Env, opts.Env...)
	logPath := b.logPath(opts.PluginID)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	if err := b.state.WritePid(opts.PluginID, pid); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	b.running[opts.PluginID] = cmd
	return nil
}

func (b *Backend) logPath(pluginID string) string {
	return filepath.Join(b.state.StateDir(), "..", "logs", pluginID, "stdout.log")
}

func (b *Backend) Stop(ctx context.Context, pluginID string) error {
	b.mu.Lock()
	cmd, ok := b.running[pluginID]
	b.mu.Unlock()
	if ok && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
		}
		b.mu.Lock()
		delete(b.running, pluginID)
		b.mu.Unlock()
		return nil
	}
	// Maybe managed by another runtime process; kill via pid file
	pid, err := b.state.ReadPid(pluginID)
	if err != nil || pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	_ = proc.Signal(syscall.SIGTERM)
	return nil
}

func (b *Backend) Delete(ctx context.Context, pluginID string) error {
	if err := b.Stop(ctx, pluginID); err != nil {
		// ignore
	}
	meta, err := b.state.LoadMeta(pluginID)
	if err == nil && meta.WorkDir != "" {
		_ = os.RemoveAll(meta.WorkDir)
	}
	return b.state.Remove(pluginID)
}

func (b *Backend) List(ctx context.Context) ([]backend.InstanceInfo, error) {
	ids, err := b.state.ListPluginIDs()
	if err != nil {
		return nil, err
	}
	var out []backend.InstanceInfo
	for _, id := range ids {
		meta, err := b.state.LoadMeta(id)
		if err != nil {
			continue
		}
		if meta.Backend != backend.BackendBinary {
			continue
		}
		pid, _ := b.state.ReadPid(id)
		status := "stopped"
		if pid > 0 {
			if proc, _ := os.FindProcess(pid); proc != nil {
				// Simple check: signal 0 to see if process is alive
				if err := proc.Signal(syscall.Signal(0)); err == nil {
					status = "running"
				}
			}
		}
		info := backend.InstanceInfo{
			PluginID: id,
			Backend:  backend.BackendBinary,
			Status:   status,
			Pid:      pid,
			WorkDir:  meta.WorkDir,
		}
		out = append(out, info)
	}
	return out, nil
}

func (b *Backend) State(ctx context.Context, pluginID string) (*backend.StateInfo, error) {
	meta, err := b.state.LoadMeta(pluginID)
	if err != nil {
		return nil, err
	}
	if meta.Backend != backend.BackendBinary {
		return nil, fmt.Errorf("plugin %s is not binary backend", pluginID)
	}
	pid, _ := b.state.ReadPid(pluginID)
	status := "stopped"
	if pid > 0 {
		if proc, _ := os.FindProcess(pid); proc != nil {
			if err := proc.Signal(syscall.Signal(0)); err == nil {
				status = "running"
			}
		}
	}
	return &backend.StateInfo{
		PluginID: pluginID,
		Backend:  backend.BackendBinary,
		Status:   status,
		Pid:      pid,
		WorkDir:  meta.WorkDir,
	}, nil
}

func (b *Backend) Log(ctx context.Context, pluginID string, opts backend.LogOptions) (io.Reader, error) {
	p := b.logPath(pluginID)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	if opts.Length <= 0 {
		return f, nil
	}
	// Simple impl: read last N lines (can be improved with ring buffer later)
	return f, nil
}

// UnregisterRunning removes the plugin from the local running map (when daemon exits).
func (b *Backend) UnregisterRunning(pluginID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.running, pluginID)
}

// IsRunning returns whether this process is managing the plugin.
func (b *Backend) IsRunning(pluginID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.running[pluginID]
	return ok
}
