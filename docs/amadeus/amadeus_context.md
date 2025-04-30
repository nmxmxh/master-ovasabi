# Amadeus Context File

This file provides continuous context about the Amadeus Knowledge Graph system for AI assistants
working with the OVASABI platform.

## System Definition

Amadeus is the knowledge persistence system for the OVASABI platform, providing a comprehensive and
programmatically accessible knowledge graph of all system components and their relationships. It
serves as both documentation and a runtime-accessible knowledge repository that evolves with the
system.

## Core Capabilities

- **Self-documenting architecture**: System components register their capabilities automatically
- **Knowledge persistence**: Maintains a persistent, evolving model of the system
- **Programmatic accessibility**: Makes system knowledge accessible to both humans and services
- **Impact analysis**: Identifies affected components before changes
- **Architectural compliance**: Enforces architectural principles
- **Visualization generation**: Auto-generates visual representations from system knowledge
- **Decision intelligence**: Provides insights for architectural decisions

## System Components

- **Knowledge Graph Store** (`amadeus/knowledge_graph.json`): JSON-based data store
- **Knowledge Graph API** (`amadeus/pkg/kg`): Go package for programmatic access
- **CLI Tool** (`amadeus/cmd/kgcli`): Command-line interface for knowledge graph access
- **Nexus Pattern** (`amadeus/nexus/pattern`): Integration with Nexus orchestration
- **Service Hooks** (`amadeus/examples`): Integration points for services

## Knowledge Graph Structure

The knowledge graph is structured with these main sections:

- `system_components`: High-level system architecture components
- `repository_structure`: Code organization and structure
- `services`: Service descriptions, capabilities, and relationships
- `nexus`: Nexus orchestration system components
- `patterns`: Pattern descriptions and compositions
- `database_practices`: Database usage patterns and schema information
- `redis_practices`: Redis usage patterns and data structures
- `amadeus_integration`: Self-description of the knowledge graph system

## Integration Methods

Services can integrate with Amadeus via:

1. **Service Hooks**: Used at service startup and during runtime
2. **Nexus Patterns**: For system-wide knowledge operations
3. **CLI Tools**: For manual and CI/CD operations
4. **Webhook API**: For external system integration

## Update Mechanisms

The knowledge graph is kept up-to-date through:

- **Service lifecycle hooks**: Updates during service startup/runtime
- **CI/CD integration**: Automated updates during deployments
- **Webhook-based updates**: External system integration
- **Scheduled jobs**: Regular validation and scanning
- **Manual updates**: CLI or direct API updates when needed

## AI & Data Science Integration

Amadeus enables:

- **Machine Learning Foundation**: Structured knowledge for AI model training
- **Decision Support Systems**: AI-assisted architectural decisions
- **Anomaly Detection**: Identifying architectural anomalies
- **Pattern Recognition**: Data analysis to identify architectural patterns
- **Evolution Tracking**: Historical analysis of system changes
- **Technical Debt Quantification**: Statistical analysis to identify refactoring needs

## Development State

- **Core Components**: Knowledge Graph Store, API, and CLI tool implemented
- **Integration Points**: Service Hooks and Nexus Pattern available
- **Documentation**: Implementation guide, integration examples, and architecture docs complete
- **Visualization**: Mermaid-based diagram generation implemented

## Evolution Tracking

The knowledge graph maintains its own evolution history:

- **Version field**: Explicit version of the knowledge graph format
- **Last updated timestamp**: When the graph was last modified
- **Backups**: Historical versions stored in `amadeus/backups`

## Implementation Status

- Basic implementation complete
- Service hooks functional
- CLI tool available
- Nexus integration established
- Documentation published
- Backup system set up

## Usage Guidelines

1. Services should register with Amadeus at startup
2. Service capabilities and dependencies should be tracked
3. CI/CD pipelines should validate knowledge graph consistency
4. Pattern implementations should be documented in the graph
5. Impact analysis should be performed before major changes

## Future Development

- Real-time knowledge graph updates via event streams
- AI-assisted system analysis
- Specialized knowledge graph query language
- Advanced visualization capabilities
- System evolution tracking and prediction

## References

For detailed information, see:

- [Implementation Guide](implementation_guide.md)
- [Integration Examples](integration_examples.md)
- [Architecture Overview](architecture.md)
- [API Reference](api_reference.md)
- [Consistent Update Guide](consistent_updates.md)
