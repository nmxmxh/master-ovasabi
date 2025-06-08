# metaversion (v0.0.1)

Canonical versioning and feature flag management for the INOS platform.

## Purpose

- Provides a single source of truth for all versioning, environment, and feature flag metadata.
- Enables context-driven, metadata-enriched, and orchestration-friendly workflows.
- Integrates with `graceful` (for orchestration) and `auth` (for authentication/session).
- Ensures all metadata includes a `versioning` field as required by platform standards.

## Features

- `Versioning` struct: Canonical version/flag metadata (system, service, user, env, flags, AB test,
  migration).
- Context helpers: Inject/extract versioning from `context.Context`.
- Metadata helpers: Merge versioning into metadata maps for DB, JWT, or API.
- Feature flag evaluation: Provider-agnostic (OpenFeature interface).
- AB test assignment: Deterministic, hash-based.
- Validation: Semantic version and required field checks.
- HTTP middleware: Injects and validates versioning for all requests.

## Initial Version

- All version fields default to `0.0.1`.

## Usage

### 1. Injecting Versioning into Context

```go
v := metaversion.NewDefault()
ctx := metaversion.InjectContext(ctx, v)
```

### 2. Extracting Versioning from Context

```go
v, ok := metaversion.FromContext(ctx)
if !ok {
    v = metaversion.NewDefault()
}
```

### 3. Merging Versioning into Metadata

```go
metadata := map[string]interface{}{}
v := metaversion.NewDefault()
metadata = metaversion.MergeMetadata(metadata, v)
```

### 4. HTTP Middleware

```go
evaluator := metaversion.NewOpenFeatureEvaluator([]string{"new_ui", "beta_api"})
http.Handle("/api", metaversion.Middleware(evaluator)(myHandler))
```

### 5. Integration with graceful

```go
// On success:
metadata = metaversion.MergeMetadata(metadata, versioning)
success := graceful.WrapSuccess(ctx, codes.OK, "operation succeeded", result, nil)
success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
    Metadata: metadata, // Now guaranteed to have versioning
    // ...
})
```

### 6. Integration with auth

```go
// On login/auth:
flags, _ := evaluator.EvaluateFlags(ctx, userID)
versioning := metaversion.Versioning{
    SystemVersion:  metaversion.InitialVersion,
    ServiceVersion: metaversion.InitialVersion,
    UserVersion:    metaversion.InitialVersion,
    Environment:    "dev",
    FeatureFlags:   flags,
    ABTestGroup:    evaluator.AssignABTest(userID),
    LastMigratedAt: metaversion.NowUTC(),
}
authCtx := &auth.AuthContext{
    UserID: userID,
    Roles: roles,
    Metadata: versioning.ToMap(),
}
ctx = auth.NewContext(ctx, authCtx)
```

## Best Practices

- Always use `metaversion` to manage and validate the `versioning` field in all metadata.
- Always propagate versioning/flags through context, using the same patterns as `auth` and
  `graceful`.
- Always validate versioning before orchestration (success/error) using `metaversion` helpers.
- Reference this package in all onboarding and implementation guides.

## License

Copyright Ovasabi Studios. All rights reserved.
