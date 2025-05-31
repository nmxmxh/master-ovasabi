package metaversion

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Middleware returns an HTTP middleware that injects and validates Versioning in the request context.
func Middleware(evaluator Evaluator, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			flags, err := evaluator.EvaluateFlags(r.Context(), userID)
			if err != nil {
				if log != nil {
					log.Error("failed to evaluate feature flags", zap.Error(err))
				}
			}
			abGroup := evaluator.AssignABTest(userID)

			versioning := Versioning{
				SystemVersion:  InitialVersion,
				ServiceVersion: InitialVersion,
				UserVersion:    InitialVersion,
				Environment:    "dev",
				FeatureFlags:   flags,
				ABTestGroup:    abGroup,
				LastMigratedAt: NowUTC(),
			}

			if err := ValidateVersioning(versioning); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				if _, writeErr := w.Write([]byte("invalid versioning: " + err.Error())); writeErr != nil {
					if log != nil {
						log.Error("failed to write error response", zap.Error(writeErr))
					}
				}
				return
			}

			ctx := InjectContext(r.Context(), versioning)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NowUTC returns the current time in UTC. Used for testability.
func NowUTC() time.Time {
	return time.Now().UTC()
}
