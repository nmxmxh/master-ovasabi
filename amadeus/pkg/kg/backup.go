package kg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupInfo contains metadata about a backup.
type BackupInfo struct {
	Timestamp   time.Time `json:"timestamp"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
}

// Backup creates a backup of the knowledge graph.
func (kg *KnowledgeGraph) Backup(description string) (*BackupInfo, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	if !kg.loaded {
		return nil, fmt.Errorf("knowledge graph not loaded")
	}

	// Create backup directory if it doesn't exist
	backupDir := "amadeus/backups"
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create timestamp-based filename
	timestamp := time.Now().UTC()
	backupFile := fmt.Sprintf("knowledge_graph_%s.json", timestamp.Format("20060102_150405"))
	backupPath := filepath.Join(backupDir, backupFile)

	// Marshal the knowledge graph to JSON
	data, err := kg.marshalWithIndent()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal knowledge graph: %w", err)
	}

	// Write to backup file
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", err)
	}

	// Create backup info
	info := &BackupInfo{
		Timestamp:   timestamp,
		Version:     kg.Version,
		Description: description,
		FilePath:    backupPath,
	}

	return info, nil
}

// ListBackups returns a list of available backups.
func ListBackups() ([]*BackupInfo, error) {
	backupDir := "amadeus/backups"

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Read directory
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]*BackupInfo, 0, len(files))
	for _, file := range files {
		// Skip directories and non-JSON files
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Parse timestamp from filename
		// Expected format: knowledge_graph_20060102_150405.json
		const timeLayout = "knowledge_graph_20060102_150405.json"
		timestamp, err := time.Parse(timeLayout, file.Name())
		if err != nil {
			// Not a knowledge graph backup file
			continue
		}

		// Create backup info
		info := &BackupInfo{
			Timestamp: timestamp,
			FilePath:  filepath.Join(backupDir, file.Name()),
		}

		// Try to read version from file
		if backup, err := LoadFromFile(info.FilePath); err == nil {
			info.Version = backup.Version
		}

		backups = append(backups, info)
	}

	return backups, nil
}

// RestoreFromBackup restores the knowledge graph from a backup file.
func (kg *KnowledgeGraph) RestoreFromBackup(backupPath string) error {
	backup, err := LoadFromFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to load backup: %w", err)
	}

	kg.mu.Lock()
	defer kg.mu.Unlock()

	// Copy all fields from backup
	kg.Version = backup.Version
	kg.LastUpdated = backup.LastUpdated
	kg.SystemComponents = backup.SystemComponents
	kg.RepositoryStructure = backup.RepositoryStructure
	kg.Services = backup.Services
	kg.Nexus = backup.Nexus
	kg.Patterns = backup.Patterns
	kg.DatabasePractices = backup.DatabasePractices
	kg.RedisPractices = backup.RedisPractices
	kg.AmadeusIntegration = backup.AmadeusIntegration
	kg.loaded = true

	return nil
}

// LoadFromFile loads a knowledge graph from a file.
func LoadFromFile(filePath string) (*KnowledgeGraph, error) {
	kg := &KnowledgeGraph{}
	if err := kg.Load(filePath); err != nil {
		return nil, err
	}
	return kg, nil
}

// Helper function to marshal the knowledge graph with indentation.
func (kg *KnowledgeGraph) marshalWithIndent() ([]byte, error) {
	return json.MarshalIndent(kg, "", "  ")
}
