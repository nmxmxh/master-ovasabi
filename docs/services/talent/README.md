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

## Translators as Talent

Translators are tracked as talent profiles, with language pairs, expertise, ratings, and booking
history. When a human translation is performed, the `translator_id` in the `translation_provenance`
field (see localization/content metadata) should reference the relevant talent profile. See
[Amadeus context](../../amadeus/amadeus_context.md#machine-vs-human-translation--translator-roles)
and [General Metadata Documentation](../metadata.md) for schema and best practices.
