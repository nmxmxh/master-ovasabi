# AI Python Module Structure

This directory has been restructured for better organization and maintainability.

## Directory Structure

```
ai/python/
├── core/               # Core application modules
│   ├── __init__.py
│   ├── main.py        # Main application entrypoint
│   ├── cli.py         # Command-line interface
│   └── llm_registry.py # LLM adapters and registry
├── db/                 # Database-related modules
│   ├── __init__.py
│   ├── db.py          # PostgreSQL connection and ORM
│   ├── vector_db.py   # Vector database interface
│   ├── vector_db_registry.py # Vector DB adapters
│   └── ai_web.py      # Web knowledge table interface
├── ai/                 # AI-specific modules
├── bus/               # Event bus and messaging
├── cognition/         # Cognitive processing
├── common/            # Common utilities and protobuf
├── crawler/           # Web crawling and knowledge extraction
├── inference/         # Model inference engines
├── knowledge/         # Knowledge graph management
├── models/            # Model files and cache
├── nexus/             # Nexus event system
├── orchestrator/      # Orchestration and coordination
├── tests/             # Test suites
├── utils/             # Utility functions
├── main.py            # Entry point (backward compatibility)
├── cli.py             # CLI entry point (backward compatibility)
└── requirements*.txt  # Python dependencies
```

## Key Changes

1. **Database modules** moved to `db/` folder:
   - `db.py` - PostgreSQL tables and connections
   - `vector_db.py` - Vector database interface
   - `vector_db_registry.py` - Vector DB adapters
   - `ai_web.py` - Web knowledge table

2. **Core application** moved to `core/` folder:
   - `main.py` - Main application entrypoint
   - `cli.py` - Command-line interface
   - `llm_registry.py` - LLM adapters and registry

3. **Backward compatibility** maintained:
   - Root-level `main.py` and `cli.py` import from core modules
   - All imports updated to use new module paths

## Usage

The restructuring is transparent to users. You can still run:

```bash
python main.py          # Main application
python cli.py --help    # CLI interface
```

All import statements have been updated to use the new module structure.
