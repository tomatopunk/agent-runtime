package runtime

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/backend/binary"
	"github.com/tomatopunk/agent-runtime/internal/daemon"
	"github.com/tomatopunk/agent-runtime/internal/state"
)

// Run starts the plugin and enters daemon monitoring (restart on crash) until stop or signal.
func (r *Runtime) Run(ctx context.Context, backendName string, opts backend.RunOptions) error {
	if err := r.state.EnsureStateDir(); err != nil {
		return err
	}
	be, err := r.backendForName(backendName)
	if err != nil {
		return err
	}
	meta := state.Meta{
		PluginID:   opts.PluginID,
		Backend:    backendName,
		RootDir:    r.rootDir,
		WorkDir:    opts.WorkDir,
		Config:     opts.Config,
		CPU:        opts.CPU,
		Mem:        opts.Mem,
		Env:        opts.Env,
		RuntimePid: os.Getpid(),
	}
	if err := r.state.Register(meta); err != nil {
		return err
	}
	if err := be.Run(ctx, opts); err != nil {
		_ = r.state.Remove(opts.PluginID)
		return err
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-sigCh
		_ = r.state.RequestStop(opts.PluginID)
		_ = be.Stop(ctx, opts.PluginID)
		cancel()
	}()
	daemon.Monitor(ctx, opts.PluginID, r.state, be, 3*time.Second)
	if b, ok := be.(*binary.Backend); ok {
		b.UnregisterRunning(opts.PluginID)
	}
	return nil
}
