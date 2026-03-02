package backend

import (
	"context"
	"io"
	"time"
)

// Backend is the runtime backend interface implemented by binary and runc.
// It wraps the executable (like containerd-shim); executables are assumed to be long-running, lifecycle is managed by the caller.
type Backend interface {
	// Run starts the plugin and returns immediately; the process/container keeps running.
	Run(ctx context.Context, opts RunOptions) error
	// Wait blocks until the plugin process/container exits or ctx is cancelled. Used by the re-exec'd shim process.
	Wait(ctx context.Context, pluginID string) error
	// Stop stops the plugin (does not remove work dir).
	Stop(ctx context.Context, pluginID string) error
	// Delete stops the plugin and removes the work dir.
	Delete(ctx context.Context, pluginID string) error
	// List returns all plugins managed by this runtime and their status.
	List(ctx context.Context) ([]InstanceInfo, error)
	// State returns the status of a single plugin.
	State(ctx context.Context, pluginID string) (*StateInfo, error)
	// Log returns a reader for the plugin log in a unified format.
	Log(ctx context.Context, pluginID string, opts LogOptions) (io.Reader, error)
}

// RunOptions are the options for starting a plugin.
type RunOptions struct {
	PluginID      string   // injected as PLUGIN_ID env (binary + runc)
	PluginVersion string   // injected as PLUGIN_VERSION env
	DeviceId      string   // injected as DEVICE_ID env
	HostType      string   // injected as HOST_TYPE env
	HostName      string   // injected as HOST_NAME env
	RootDir       string   // runtime root dir
	WorkDir       string   // for binary: work dir (cwd); for runc: bundle path
	// Executable: host path to the binary to run. Binary backend runs it directly;
	// runc backend copies it into bundle rootfs and runs it inside the container.
	Executable string   // required
	Args       []string // optional args: binary = append to launch command; runc = args inside container
	CPU        string   // cgroup CPU quota, e.g. "0.5"
	Mem        string   // cgroup memory quota, e.g. "128m"
	Env        []string // extra KEY=VALUE env (in addition to injected vars)
}

// InstanceInfo is a plugin summary for list output.
type InstanceInfo struct {
	PluginID  string    `json:"plugin_id"`
	Backend   string    `json:"backend"` // "binary" | "runc"
	Status    string    `json:"status"`  // "running" | "stopped" | "unknown"
	Pid       int       `json:"pid,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	WorkDir   string    `json:"work_dir,omitempty"`
}

// StateInfo is the state of a single plugin for state output.
type StateInfo struct {
	PluginID   string    `json:"plugin_id"`
	Backend    string    `json:"backend"`
	Status     string    `json:"status"`
	Pid        int       `json:"pid,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	ExitStatus int       `json:"exit_status,omitempty"`
	WorkDir    string    `json:"work_dir,omitempty"`
}

// LogOptions are options for reading logs.
type LogOptions struct {
	Start  *time.Time
	End    *time.Time
	Length int    // max number of lines, 0 = no limit
	Format string // "json" | "text"
}

const (
	BackendBinary = "binary"
	BackendRunc   = "runc"
)
