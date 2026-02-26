package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	StopRequestedFile = "stop_requested"
	MetaFile          = "meta.json"
	PidFile           = "pid"
)

// Meta is the metadata for each plugin under the state dir.
type Meta struct {
	PluginID      string   `json:"plugin_id"`
	PluginVersion string   `json:"plugin_version,omitempty"`
	DeviceId      string   `json:"device_id,omitempty"`
	HostType      string   `json:"host_type,omitempty"`
	HostName      string   `json:"host_name,omitempty"`
	Backend       string   `json:"backend"`
	RootDir       string   `json:"root_dir"`
	WorkDir       string   `json:"work_dir"`
	Config        string   `json:"config"`
	CPU           string   `json:"cpu"`
	Mem           string   `json:"mem"`
	Env           []string `json:"env,omitempty"`
	RuntimePid    int      `json:"runtime_pid"` // pid of the runtime process that monitors this plugin
}

// Manager manages the state dir: registration, stop requests, enumeration.
type Manager struct {
	rootDir string
	mu      sync.RWMutex
}

func NewManager(rootDir string) *Manager {
	return &Manager{rootDir: rootDir}
}

func (m *Manager) StateDir() string {
	return filepath.Join(m.rootDir, "state")
}

func (m *Manager) PluginDir(pluginID string) string {
	return filepath.Join(m.StateDir(), pluginID)
}

func (m *Manager) EnsureStateDir() error {
	return os.MkdirAll(m.StateDir(), 0755)
}

// Register creates the plugin dir under state and writes meta.
func (m *Manager) Register(meta Meta) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	dir := m.PluginDir(meta.PluginID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, MetaFile), b, 0644)
}

// RequestStop writes a stop request file; the monitor process will detect it and exit.
func (m *Manager) RequestStop(pluginID string) error {
	path := filepath.Join(m.PluginDir(pluginID), StopRequestedFile)
	return os.WriteFile(path, []byte("1"), 0644)
}

// StopRequested returns whether a stop has been requested for the plugin.
func (m *Manager) StopRequested(pluginID string) bool {
	path := filepath.Join(m.PluginDir(pluginID), StopRequestedFile)
	_, err := os.Stat(path)
	return err == nil
}

// LoadMeta reads the plugin meta.
func (m *Manager) LoadMeta(pluginID string) (*Meta, error) {
	path := filepath.Join(m.PluginDir(pluginID), MetaFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta Meta
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// ListPluginIDs returns all plugin-id dirs under state.
func (m *Manager) ListPluginIDs() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	dir := m.StateDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// Remove removes the plugin state dir (used by delete).
func (m *Manager) Remove(pluginID string) error {
	return os.RemoveAll(m.PluginDir(pluginID))
}

// WritePid writes the plugin process pid (used by binary backend).
func (m *Manager) WritePid(pluginID string, pid int) error {
	path := filepath.Join(m.PluginDir(pluginID), PidFile)
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// ReadPid reads the pid file (numeric string).
func (m *Manager) ReadPid(pluginID string) (int, error) {
	path := filepath.Join(m.PluginDir(pluginID), PidFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var pid int
	_, _ = fmt.Sscanf(string(b), "%d", &pid)
	return pid, nil
}
