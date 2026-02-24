package daemon

import (
	"context"
	"time"

	"github.com/tomatopunk/agent-runtime/internal/backend"
	"github.com/tomatopunk/agent-runtime/internal/state"
)

// Monitor watches a single plugin: if not stop-requested and process/container has exited, restart it.
func Monitor(
	ctx context.Context,
	pluginID string,
	stateManager *state.Manager,
	be backend.Backend,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if stateManager.StopRequested(pluginID) {
				return
			}
			info, err := be.State(ctx, pluginID)
			if err != nil {
				continue
			}
			if info.Status == "running" {
				continue
			}
			// Stopped and not stop-requested -> restart
			meta, err := stateManager.LoadMeta(pluginID)
			if err != nil {
				continue
			}
			opts := backend.RunOptions{
				PluginID: meta.PluginID,
				RootDir:  meta.RootDir,
				WorkDir:  meta.WorkDir,
				Config:   meta.Config,
				CPU:      meta.CPU,
				Mem:      meta.Mem,
				Env:      meta.Env,
			}
			_ = be.Run(ctx, opts)
		}
	}
}
