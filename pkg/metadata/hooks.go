package metadata

import (
	"context"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
)

type Hook interface {
	PreUpdate(ctx context.Context, oldMeta, newMeta *commonpb.Metadata) error
	PostUpdate(ctx context.Context, meta *commonpb.Metadata)
}

var registeredHooks []Hook

// RegisterMetadataHook registers a new metadata hook.
func RegisterMetadataHook(hook Hook) {
	registeredHooks = append(registeredHooks, hook)
}

// RunPreUpdateHooks runs all registered PreUpdate hooks.
func RunPreUpdateHooks(ctx context.Context, oldMeta, newMeta *commonpb.Metadata) error {
	for _, hook := range registeredHooks {
		if err := hook.PreUpdate(ctx, oldMeta, newMeta); err != nil {
			return err
		}
	}
	return nil
}

// RunPostUpdateHooks runs all registered PostUpdate hooks.
func RunPostUpdateHooks(ctx context.Context, meta *commonpb.Metadata) {
	for _, hook := range registeredHooks {
		hook.PostUpdate(ctx, meta)
	}
}
