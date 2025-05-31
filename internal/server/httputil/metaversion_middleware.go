package httputil

import (
	"net/http"

	"github.com/nmxmxh/master-ovasabi/pkg/metaversion"
	"go.uber.org/zap"
)

// MetaversionMiddleware returns an HTTP middleware that injects versioning into context for every request.
func MetaversionMiddleware(evaluator metaversion.Evaluator, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			flags, err := evaluator.EvaluateFlags(r.Context(), userID)
			if err != nil {
				logger.Warn("Failed to evaluate feature flags", zap.Error(err), zap.String("user_id", userID))
			}
			abGroup := evaluator.AssignABTest(userID)

			versioning := metaversion.Versioning{
				SystemVersion:  metaversion.InitialVersion,
				ServiceVersion: metaversion.InitialVersion,
				UserVersion:    metaversion.InitialVersion,
				Environment:    "dev",
				FeatureFlags:   flags,
				ABTestGroup:    abGroup,
				LastMigratedAt: metaversion.NowUTC(),
			}

			if err := metaversion.ValidateVersioning(versioning); err != nil {
				logger.Warn("Invalid versioning metadata", zap.Error(err), zap.String("user_id", userID))
				w.WriteHeader(http.StatusBadRequest)
				if _, writeErr := w.Write([]byte("invalid versioning: " + err.Error())); writeErr != nil {
					logger.Error("Failed to write error response", zap.Error(writeErr))
				}
				return
			}

			logger.Debug("Injected versioning", zap.Any("versioning", versioning), zap.String("user_id", userID))
			ctx := metaversion.InjectContext(r.Context(), versioning)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
