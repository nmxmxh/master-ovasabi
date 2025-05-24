# utils/safeint.go — Safe Integer Conversion Helpers

## Overview

The `safeint` helpers provide safe conversion from Go's `int` type to `int32`, `int64`, and now
arbitrary-precision `*big.Int`, clamping values to the target type's range where appropriate. This
prevents integer overflow bugs and satisfies static analysis tools (e.g., gosec G115).

## Why Use These Helpers?

- Prevents integer overflow when converting from `int` to `int32`/`int64` (especially on 64-bit
  systems).
- Supports future-proofing for scientific, financial, or blockchain use cases with `*big.Int`.
- Satisfies linters and security tools (e.g., gosec G115).
- Keeps code DRY, readable, and consistent across the codebase.

## Usage

```go
import "github.com/nmxmxh/master-ovasabi/pkg/utils"

page32 := utils.ToInt32(page)
pageSize64 := utils.ToInt64(pageSize)
bigVal := utils.ToBigInt(page)
bigVal64 := utils.ToBigInt64(someInt64)
```

## Example

Instead of:

```go
page32 := int32(page)
bigVal := big.NewInt(int64(page))
```

Use:

```go
page32 := utils.ToInt32(page)
bigVal := utils.ToBigInt(page)
```

## When to Use

- Any time you convert from `int` to `int32`, `int64`, or `*big.Int` for APIs, Protobufs, database
  calls, or scientific/financial calculations.
- When you see linter warnings about integer overflow (e.g., gosec G115).
- When you need arbitrary-precision math (e.g., scientific, blockchain, or very large counters).

## Implementation

See `safeint.go` for implementation details.
