package runc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/state"
)

// copyExecutableToRootfs copies the host binary into bundle rootfs so runc can run it inside the container.
func copyExecutableToRootfs(workDir, hostExecutable string) error {
	src, err := os.Open(hostExecutable)
	if err != nil {
		return fmt.Errorf("open executable %s: %w", hostExecutable, err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("executable %s is not a regular file", hostExecutable)
	}
	// rootfs/app/plugin
	destDir := filepath.Join(workDir, "rootfs", "app")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	destPath := filepath.Join(destDir, "plugin")
	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	_, err = io.Copy(dest, src)
	dest.Close()
	if err != nil {
		os.Remove(destPath)
		return err
	}
	return nil
}

// runcState is (partial) output of runc state
type runcState struct {
	Pid    int    `json:"pid"`
	Status string `json:"status"`
}

// Backend runs runc internally; unified state/list/log.
type Backend struct {
	state    *state.Manager
	runcPath string
	mu       sync.Mutex
	cancels  map[string]context.CancelFunc
}

func New(stateManager *state.Manager, runcPath string) *Backend {
	if runcPath == "" {
		runcPath = "runc"
	}
	return &Backend{
		state:    stateManager,
		runcPath: runcPath,
		cancels:  make(map[string]context.CancelFunc),
	}
}

// Path inside container where the executable is copied (under rootfs).
const inContainerExePath = "/app/plugin"

func (b *Backend) Run(ctx context.Context, opts backend.RunOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if opts.PluginID == "" || opts.WorkDir == "" {
		return fmt.Errorf("plugin_id and work_dir (bundle path) are required")
	}
	if opts.Executable == "" {
		return fmt.Errorf("executable (host path to binary) is required for runc")
	}
	if err := os.MkdirAll(opts.WorkDir, 0755); err != nil {
		return err
	}
	// Copy host executable into bundle rootfs so container can run it
	if err := copyExecutableToRootfs(opts.WorkDir, opts.Executable); err != nil {
		return err
	}
	if err := writeConfigJSON(opts.WorkDir, opts); err != nil {
		return err
	}
	logPath := b.logPath(opts.PluginID)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, b.runcPath, "run", opts.PluginID)
	cmd.Dir = opts.WorkDir
	cmd.Env = os.Environ()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	go func() {
		defer logFile.Close()
		_ = cmd.Run()
	}()
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (b *Backend) logPath(pluginID string) string {
	return filepath.Join(b.state.StateDir(), "..", "logs", pluginID, "stdout.log")
}

func (b *Backend) Stop(ctx context.Context, pluginID string) error {
	meta, err := b.state.LoadMeta(pluginID)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, b.runcPath, "delete", "--force", pluginID)
	cmd.Dir = meta.WorkDir
	_ = cmd.Run()
	return nil
}

func (b *Backend) Delete(ctx context.Context, pluginID string) error {
	_ = b.Stop(ctx, pluginID)
	meta, err := b.state.LoadMeta(pluginID)
	if err == nil && meta.WorkDir != "" {
		_ = os.RemoveAll(meta.WorkDir)
	}
	return b.state.Remove(pluginID)
}

func (b *Backend) getRuncState(pluginID string) (*runcState, error) {
	meta, err := b.state.LoadMeta(pluginID)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(b.runcPath, "state", pluginID)
	cmd.Dir = meta.WorkDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var s runcState
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, err
	}
	return &s, nil
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
		if meta.Backend != backend.BackendRunc {
			continue
		}
		rs, err := b.getRuncState(id)
		if err != nil {
			out = append(out, backend.InstanceInfo{
				PluginID: id,
				Backend:  backend.BackendRunc,
				Status:   "stopped",
				WorkDir:  meta.WorkDir,
			})
			continue
		}
		status := strings.ToLower(rs.Status)
		if status == "" {
			status = "running"
		}
		out = append(out, backend.InstanceInfo{
			PluginID: id,
			Backend:  backend.BackendRunc,
			Status:   status,
			Pid:      rs.Pid,
			WorkDir:  meta.WorkDir,
		})
	}
	return out, nil
}

func (b *Backend) State(ctx context.Context, pluginID string) (*backend.StateInfo, error) {
	meta, err := b.state.LoadMeta(pluginID)
	if err != nil {
		return nil, err
	}
	if meta.Backend != backend.BackendRunc {
		return nil, fmt.Errorf("plugin %s is not runc backend", pluginID)
	}
	rs, err := b.getRuncState(pluginID)
	if err != nil {
		return &backend.StateInfo{
			PluginID: pluginID,
			Backend:  backend.BackendRunc,
			Status:   "stopped",
			WorkDir:  meta.WorkDir,
		}, nil
	}
	status := strings.ToLower(rs.Status)
	if status == "" {
		status = "running"
	}
	return &backend.StateInfo{
		PluginID: pluginID,
		Backend:  backend.BackendRunc,
		Status:   status,
		Pid:      rs.Pid,
		WorkDir:  meta.WorkDir,
	}, nil
}

func (b *Backend) Log(ctx context.Context, pluginID string, opts backend.LogOptions) (io.Reader, error) {
	p := b.logPath(pluginID)
	return os.Open(p)
}

// RegisterCancel registers the run's cancel for use on stop.
func (b *Backend) RegisterCancel(pluginID string, cancel context.CancelFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cancels[pluginID] = cancel
}

func (b *Backend) UnregisterCancel(pluginID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.cancels, pluginID)
}

// Wait blocks until the container exits or ctx is cancelled (used by re-exec'd shim).
func (b *Backend) Wait(ctx context.Context, pluginID string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			rs, err := b.getRuncState(pluginID)
			if err != nil || rs.Status == "" || strings.ToLower(rs.Status) == "stopped" {
				return nil
			}
		}
	}
}
