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
	// container id -> cancel for runs started by this process
	cancels map[string]context.CancelFunc
}

func New(stateManager *state.Manager, runcPath string) *Backend {
	if runcPath == "" {
		runcPath = "runc"
	}
	return &Backend{state: stateManager, runcPath: runcPath, cancels: make(map[string]context.CancelFunc)}
}

func (b *Backend) Run(ctx context.Context, opts backend.RunOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if opts.PluginID == "" || opts.WorkDir == "" {
		return fmt.Errorf("plugin_id and work_dir (bundle path) are required")
	}
	logPath := b.logPath(opts.PluginID)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	// Run runc run in a goroutine without --detach; write container stdout/stderr to unified log
	cmd := exec.CommandContext(ctx, b.runcPath, "run", opts.PluginID)
	cmd.Dir = opts.WorkDir
	cmd.Env = opts.Env
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	go func() {
		defer logFile.Close()
		_ = cmd.Run()
	}()
	// Wait for runc to start before returning so daemon's first check does not think it is stopped
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
			PluginID:  id,
			Backend:   backend.BackendRunc,
			Status:    status,
			Pid:       rs.Pid,
			WorkDir:   meta.WorkDir,
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
		PluginID:  pluginID,
		Backend:   backend.BackendRunc,
		Status:    status,
		Pid:       rs.Pid,
		WorkDir:   meta.WorkDir,
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

// WaitUntilStopped polls runc state until the container is not running (for daemon restart).
func (b *Backend) WaitUntilStopped(pluginID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		rs, err := b.getRuncState(pluginID)
		if err != nil || rs.Status == "" || strings.ToLower(rs.Status) == "stopped" {
			return
		}
	}
}
