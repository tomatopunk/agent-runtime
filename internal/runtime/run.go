package runtime

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/state"
)

// Run starts the plugin and returns immediately; lifecycle is managed by the caller (e.g. stop via separate CLI or upper layer).
func (r *Runtime) Run(ctx context.Context, backendName string, opts backend.RunOptions) error {
	if err := r.state.EnsureStateDir(); err != nil {
		return err
	}
	be, err := r.backendForName(backendName)
	if err != nil {
		return err
	}
	meta := state.Meta{
		PluginID:      opts.PluginID,
		PluginVersion: opts.PluginVersion,
		DeviceId:      opts.DeviceId,
		HostType:      opts.HostType,
		HostName:      opts.HostName,
		Backend:       backendName,
		RootDir:       r.rootDir,
		WorkDir:       opts.WorkDir,
		Executable:    opts.Executable,
		Args:          opts.Args,
		CPU:           opts.CPU,
		Mem:           opts.Mem,
		Env:           opts.Env,
		RuntimePid:    os.Getpid(),
	}
	if err := r.state.Register(meta); err != nil {
		return err
	}
	return be.Run(ctx, opts)
}

// RunAndWait starts the plugin and blocks until it exits or SIGTERM/SIGINT (used by the re-exec'd shim; keeps plugin as child, no orphan).
func (r *Runtime) RunAndWait(ctx context.Context, backendName string, opts backend.RunOptions) error {
	if err := r.Run(ctx, backendName, opts); err != nil {
		return err
	}
	be, _ := r.backendForName(backendName)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-sigCh
		_ = r.state.RequestStop(opts.PluginID)
		_ = be.Stop(ctx, opts.PluginID)
		cancel()
	}()
	return be.Wait(ctx, opts.PluginID)
}
