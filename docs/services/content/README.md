# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14

> **Standard:** This service follows the
> [Unified Communication & Calculation Standard](../../amadeus/amadeus_context.md#unified-communication--calculation-standard-grpc-rest-websocket-and-metadata-driven-orchestration).
>
> - Exposes calculation/enrichment endpoints using canonical metadata
> - Documents all metadata fields and calculation chains
> - References the Amadeus context and unified standard
> - Uses Makefile/Docker for all builds and proto generation

## Translation Provenance & Optimizations

All localized content must set the `translation_provenance` field in
`metadata.service_specific.content`, distinguishing between machine and human translation.
Optimizations and reviews by translators should be tracked in `optimizations`. See
[Amadeus context](../../amadeus/amadeus_context.md#machine-vs-human-translation--translator-roles)
and [General Metadata Documentation](../metadata.md) for schema and best practices.
