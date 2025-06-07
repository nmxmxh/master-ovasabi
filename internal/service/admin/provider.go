package admin

import (
	"context"
	"database/sql"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	service "github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the admin service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, redisProvider, log, eventEnabled. masterRepo and provider are unused.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter EventEmitter,
	db *sql.DB,
	_ repository.MasterRepository, // unused
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{}, // unused, keep for signature consistency
) error {
	masterRepoLocal := repository.NewRepository(db, log)
	repo := NewRepository(db, masterRepoLocal)
	cache, err := redisProvider.GetCache(ctx, "admin")
	if err != nil {
		log.With(zap.String("service", "admin")).Warn("Failed to get admin cache", zap.Error(err), zap.String("cache", "admin"), zap.String("context", ctxValue(ctx)))
	}
	// TODO: If admin needs to call user service, use event bus or create a gRPC client on demand
	adminService := NewService(log, repo, nil, cache, eventEmitter, eventEnabled)
	if err := container.Register((*adminpb.AdminServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return adminService, nil
	}); err != nil {
		log.With(zap.String("service", "admin")).Error("Failed to register admin service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		hello.StartHelloWorldLoop(ctx, prov, log, "admin")
	}
	return nil
}

// ctxValue extracts a string for logging from context (e.g., request ID or trace ID).
func ctxValue(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
