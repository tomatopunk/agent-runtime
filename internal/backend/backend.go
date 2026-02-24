package backend

import (
	"context"
	"io"
	"time"
)

// Backend is the runtime backend interface implemented by binary and runc.
type Backend interface {
	// Run starts the plugin; the caller is responsible for daemon monitoring.
	Run(ctx context.Context, opts RunOptions) error
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
	PluginID string
	RootDir  string // runtime root dir
	WorkDir  string // for binary: work dir; for runc: bundle path
	Config   string // config path (e.g. executable path for binary)
	CPU      string // cgroup CPU quota, e.g. "0.5"
	Mem      string // cgroup memory quota, e.g. "128m"
	Env      []string
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
