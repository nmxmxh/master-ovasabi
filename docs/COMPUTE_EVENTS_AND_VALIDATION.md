# Canonical Compute Events and Validation (Nexus-Compatible)

This document defines the canonical event types and validation rules for the universal,
device-agnostic compute fabric. All messages travel via `NexusService.EmitEvent/SubscribeEvents` and
use `common.Payload` + `common.Metadata`.

## Event Types (service:action:v1:lifecycle)

- compute:dispatch:v1:requested
- compute:dispatch:v1:accepted
- compute:dispatch:v1:progress
- compute:dispatch:v1:success
- compute:dispatch:v1:failed
- compute:dispatch:v1:cancelled
- compute:capabilities:v1:update
- compute:module:v1:register (optional)
- compute:module:v1:validate (optional)

## Payload Types (in common.Payload)

- requested: `common.v1.ComputeEnvelope`
- accepted: `common.v1.ComputeAssignment` or `common.v1.ComputeClaim`
- progress: `common.v1.ComputeProgress`
- success: `common.v1.ComputeResult`
- failed: `common.v1.ComputeFailure`
- capabilities:update: `common.v1.Capability`

See `api/protos/common/v1/compute.proto` for message definitions (`DataRef`, `TensorSpec`,
`BufferSpec`, `GPUDescriptor`, `Capability`, `Requirements`, `ModuleSpec`, `ComputeEnvelope`,
`ComputeClaim`, `ComputeAssignment`, `ComputeProgress`, `ComputeResult`, `ComputeFailure`).

## Minimum Validation Rules

1. Envelope required fields
   - `task_id`, `op`, `module.kind`, `requirements.min`
2. DataRef body exclusivity
   - Exactly one of: `inline_json`, `inline_bytes`, `uri`
3. GPU requirement expression
   - If GPU required, set `requirements.min.webgpu = true` or provide `requirements.min.gpu` (with
     backend, vendor, features/limits as needed)
4. Module integrity
   - If `module.uri` is remote (http(s)/s3/ipfs), `module.hash` must be provided (sha256 or similar)
5. QoS hints (optional but recommended)
   - `requirements.qos[priority]`, `requirements.qos[deadline_ms]`, `requirements.qos[retries]`
6. Security
   - `security[task_token]` present for requester-signed tasks

## Metadata and Correlation

- Use `common.Metadata.GlobalContext.correlation_id` and/or include `correlationId` in payload for
  request/response pairing, consistent with campaign state flows.
- Include `campaign_id`, `tenant`, `user_id` in `common.Metadata` for routing.

## Capability Announcement (workers)

- On start and when changing, workers publish `compute:capabilities:v1:update` with
  `common.v1.Capability`, including:
  - WASM/threads/SIMD flags
  - CPU cores, memory
  - `GPUDescriptor` (backend, vendor, features, limits)
  - labels/attributes (browser, os, driver versions, network, battery state)

## Scheduling and Routing (summary)

- Filter candidates by `Requirements.min` (must-have)
- Score by `preferred`, QoS, locality, data proximity
- Assign via `compute:dispatch:v1:accepted` + `ComputeAssignment`
- Worker executes, emits `progress`, then `success` (or `failed`)
- Results delivered to `return_channel` if set; otherwise to requester stream

## Security & Sandbox Guidelines

- Workers must echo `security.task_token` in results for verification
- WASM executors enforce `ModuleSpec.permissions` (WASI fs/net, gpu_access, memory limits)
- Browser threads require COOP/COEP; gateway retains CORS controls

## Examples (JSON payload excerpts)

requested (ComputeEnvelope):

```json
{
  "task_id": "t-123",
  "op": "fft",
  "version": "1.0.0",
  "module": {"kind": "wasm", "uri": "https://cdn/fft.wasm", "hash": "sha256:...", "entry": "runFFT"},
  "requirements": {"min": {"webgpu": true, "gpu": {"backend": "webgpu", "features": ["shader_f16"]}}},
  "inputs": [{"name": "signal", "content_type": "application/octet-stream", "inline_bytes": "..."}],
  "params": {"window": "hann"},
  "return_channel": "analytics:ingest:v1:requested",
  "security": {"task_token": "hmac:..."}
}
```

accepted (ComputeAssignment):

```json
{"task_id": "t-123", "worker_id": "edge-42"}
```

progress (ComputeProgress):

```json
{"task_id": "t-123", "pct": 65, "metrics": {"tokens_s": "1200", "mem_used_mb": "512"}}
```

success (ComputeResult):

```json
{"task_id": "t-123", "outputs": [{"name": "spectrum", "content_type": "application/octet-stream", "uri": "s3://bucket/obj"}], "summary": {"bins": "1024"}}
```
