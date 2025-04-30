# Package redis

## Constants

### PatternOriginSystem

## Types

### ExecutorOptions

ExecutorOptions defines configuration options for the pattern executor

### OperationStep

OperationStep defines a single step in a pattern

### PatternCategory

PatternCategory defines the category of a pattern

### PatternExecutor

PatternExecutor executes stored patterns

#### Methods

##### ExecutePattern

ExecutePattern executes a pattern with the given input

### PatternOrigin

PatternOrigin defines the source of a pattern

### PatternStore

PatternStore manages pattern storage in Redis

#### Methods

##### DeletePattern

DeletePattern deletes a pattern from Redis

##### GetPattern

GetPattern retrieves a pattern from Redis

##### ListPatterns

ListPatterns lists patterns based on filters

##### StorePattern

StorePattern stores a pattern in Redis

##### UpdatePatternStats

UpdatePatternStats updates pattern usage statistics

### StoredPattern

StoredPattern represents a pattern stored in Redis
