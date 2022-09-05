package kindacool

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

func EnsurePlugins(ctx context.Context, w auto.Workspace) error {
	// for inline source programs, we must manage plugins ourselves
	err := w.InstallPlugin(ctx, "openstack", "v3.9.0")
	if err != nil {
		return fmt.Errorf("failed to install openstack plugin: %w", err)
	}

	err = w.InstallPlugin(ctx, "command", "v0.7.0")
	if err != nil {
		return fmt.Errorf("failed to install command plugin: %w", err)
	}

	return nil
}
