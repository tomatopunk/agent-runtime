package runtime

import (
	"context"
	"fmt"
	"io"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/backend/binary"
	"github.com/tomatopunk/agent-runtime/internal/backend/runc"
	"github.com/tomatopunk/agent-runtime/internal/state"
)

// Runtime is the unified facade: holds state and both backends, returns Backend by plugin or backend name.
type Runtime struct {
	rootDir string
	state   *state.Manager
	binary  backend.Backend
	runc    backend.Backend
}

// New creates a Runtime for the given rootDir.
func New(rootDir string) *Runtime {
	mgr := state.NewManager(rootDir)
	return &Runtime{
		rootDir: rootDir,
		state:   mgr,
		binary:  binary.New(mgr),
		runc:    runc.New(mgr, ""),
	}
}

// backendForName returns the Backend for the given backend name.
func (r *Runtime) backendForName(name string) (backend.Backend, error) {
	switch name {
	case backend.BackendBinary:
		return r.binary, nil
	case backend.BackendRunc:
		return r.runc, nil
	default:
		return nil, fmt.Errorf("unknown backend: %s", name)
	}
}

// BackendFor looks up meta by pluginID and returns the corresponding Backend.
func (r *Runtime) BackendFor(pluginID string) (backend.Backend, error) {
	meta, err := r.state.LoadMeta(pluginID)
	if err != nil {
		return nil, err
	}
	return r.backendForName(meta.Backend)
}

func (r *Runtime) StateManager() *state.Manager { return r.state }

// Stop requests stop and stops the plugin.
func (r *Runtime) Stop(ctx context.Context, pluginID string) error {
	_ = r.state.RequestStop(pluginID)
	be, err := r.BackendFor(pluginID)
	if err != nil {
		return err
	}
	return be.Stop(ctx, pluginID)
}

// Delete stops the plugin and cleans up.
func (r *Runtime) Delete(ctx context.Context, pluginID string) error {
	_ = r.state.RequestStop(pluginID)
	be, err := r.BackendFor(pluginID)
	if err != nil {
		return err
	}
	return be.Delete(ctx, pluginID)
}

// List returns all plugins; caller formats the output.
func (r *Runtime) List(ctx context.Context) ([]backend.InstanceInfo, error) {
	listB, _ := r.binary.List(ctx)
	listR, _ := r.runc.List(ctx)
	all := append(listB, listR...)
	if all == nil {
		all = []backend.InstanceInfo{}
	}
	return all, nil
}

// State returns a single plugin's state; caller formats the output.
func (r *Runtime) State(ctx context.Context, pluginID string) (*backend.StateInfo, error) {
	be, err := r.BackendFor(pluginID)
	if err != nil {
		return nil, err
	}
	return be.State(ctx, pluginID)
}

// Log returns a Reader for the plugin log; caller copies to stdout.
func (r *Runtime) Log(ctx context.Context, pluginID string, opts backend.LogOptions) (io.Reader, error) {
	be, err := r.BackendFor(pluginID)
	if err != nil {
		return nil, err
	}
	return be.Log(ctx, pluginID, opts)
}

// Destroy stops and removes all plugins.
func (r *Runtime) Destroy(ctx context.Context) error {
	ids, err := r.state.ListPluginIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		_ = r.state.RequestStop(id)
		be, err := r.BackendFor(id)
		if err != nil {
			continue
		}
		_ = be.Delete(ctx, id)
	}
	return nil
}
