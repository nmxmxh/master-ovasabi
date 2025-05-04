package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

func main() {
	// Define subcommands
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getPath := getCmd.String("path", "", "Path to knowledge graph node (e.g., 'services.core_services.user_service')")
	getOutput := getCmd.String("output", "json", "Output format (json|yaml|text)")

	addServiceCmd := flag.NewFlagSet("add-service", flag.ExitOnError)
	serviceCategory := addServiceCmd.String("category", "core_services", "Service category")
	serviceName := addServiceCmd.String("name", "", "Service name")
	serviceFile := addServiceCmd.String("file", "", "JSON file containing service information")

	addPatternCmd := flag.NewFlagSet("add-pattern", flag.ExitOnError)
	patternCategory := addPatternCmd.String("category", "core_patterns", "Pattern category")
	patternName := addPatternCmd.String("name", "", "Pattern name")
	patternFile := addPatternCmd.String("file", "", "JSON file containing pattern information")

	visualizeCmd := flag.NewFlagSet("visualize", flag.ExitOnError)
	visualizeFormat := visualizeCmd.String("format", "mermaid", "Visualization format (mermaid|dot|json)")
	visualizeSection := visualizeCmd.String("section", "", "Section of knowledge graph to visualize")
	visualizeOutput := visualizeCmd.String("output", "", "Output file path")

	backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)
	backupDesc := backupCmd.String("desc", "", "Description of the backup")

	listBackupsCmd := flag.NewFlagSet("list-backups", flag.ExitOnError)
	listBackupsFormat := listBackupsCmd.String("format", "text", "Output format (json|text)")

	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	restorePath := restoreCmd.String("path", "", "Path to backup file")

	// Check if no command provided
	if len(os.Args) < 2 {
		fmt.Println("Expected subcommand: get, add-service, add-pattern, visualize, backup, list-backups, or restore")
		os.Exit(1)
	}

	// Handle commands
	switch os.Args[1] {
	case "get":
		if err := getCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Error parsing get command: %v\n", err)
			os.Exit(1)
		}
		handleGetCommand(*getPath, *getOutput)
	case "add-service":
		if err := addServiceCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Error parsing add-service command: %v\n", err)
			os.Exit(1)
		}
		handleAddServiceCommand(*serviceCategory, *serviceName, *serviceFile)
	case "add-pattern":
		if err := addPatternCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Error parsing add-pattern command: %v\n", err)
			os.Exit(1)
		}
		handleAddPatternCommand(*patternCategory, *patternName, *patternFile)
	case "visualize":
		if err := visualizeCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Error parsing visualize command: %v\n", err)
			os.Exit(1)
		}
		handleVisualizeCommand(*visualizeFormat, *visualizeSection, *visualizeOutput)
	case "backup":
		if err := backupCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Error parsing backup command: %v\n", err)
			os.Exit(1)
		}
		handleBackupCommand(*backupDesc)
	case "list-backups":
		if err := listBackupsCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Error parsing list-backups command: %v\n", err)
			os.Exit(1)
		}
		handleListBackupsCommand(*listBackupsFormat)
	case "restore":
		if err := restoreCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Error parsing restore command: %v\n", err)
			os.Exit(1)
		}
		handleRestoreCommand(*restorePath)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func handleGetCommand(path, output string) {
	if path == "" {
		fmt.Println("Error: path is required")
		os.Exit(1)
	}

	// Get knowledge graph
	knowledgeGraph := kg.DefaultKnowledgeGraph()

	// Get value from knowledge graph
	value, err := knowledgeGraph.GetNode(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Output result
	switch output {
	case "json":
		jsonBytes, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
	case "yaml":
		fmt.Println("YAML output not implemented yet")
	case "text":
		fmt.Printf("%v\n", value)
	default:
		fmt.Printf("Unknown output format: %s\n", output)
		os.Exit(1)
	}
}

func handleAddServiceCommand(category, name, filePath string) {
	if name == "" {
		fmt.Println("Error: service name is required")
		os.Exit(1)
	}

	if filePath == "" {
		fmt.Println("Error: file is required")
		os.Exit(1)
	}

	// Read service information from file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Parse service information
	var serviceInfo map[string]interface{}
	if err := json.Unmarshal(data, &serviceInfo); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add service to knowledge graph
	knowledgeGraph := kg.DefaultKnowledgeGraph()
	if err := knowledgeGraph.AddService(category, name, serviceInfo); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save knowledge graph
	if err := knowledgeGraph.Save("amadeus/knowledge_graph.json"); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Service '%s' added to category '%s'\n", name, category)
}

func handleAddPatternCommand(category, name, filePath string) {
	if name == "" {
		fmt.Println("Error: pattern name is required")
		os.Exit(1)
	}

	if filePath == "" {
		fmt.Println("Error: file is required")
		os.Exit(1)
	}

	// Read pattern information from file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Parse pattern information
	var patternInfo map[string]interface{}
	if err := json.Unmarshal(data, &patternInfo); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add pattern to knowledge graph
	knowledgeGraph := kg.DefaultKnowledgeGraph()
	if err := knowledgeGraph.AddPattern(category, name, patternInfo); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save knowledge graph
	if err := knowledgeGraph.Save("amadeus/knowledge_graph.json"); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Pattern '%s' added to category '%s'\n", name, category)
}

func handleVisualizeCommand(format, section, output string) {
	knowledgeGraph := kg.DefaultKnowledgeGraph()

	// Generate visualization
	data, err := knowledgeGraph.GenerateVisualization(format, section)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Output visualization
	if output == "" {
		fmt.Println(string(data))
	} else {
		if err := os.WriteFile(output, data, 0644); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Visualization written to %s\n", output)
	}
}

func handleBackupCommand(description string) {
	if description == "" {
		// Default description
		description = fmt.Sprintf("Manual backup created on %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	// Create backup
	knowledgeGraph := kg.DefaultKnowledgeGraph()
	info, err := knowledgeGraph.Backup(description)
	if err != nil {
		fmt.Printf("Error creating backup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Backup created successfully:\n")
	fmt.Printf("  Timestamp: %s\n", info.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Version: %s\n", info.Version)
	fmt.Printf("  Description: %s\n", info.Description)
	fmt.Printf("  File path: %s\n", info.FilePath)
}

func handleListBackupsCommand(format string) {
	// Get list of backups
	backups, err := kg.ListBackups()
	if err != nil {
		fmt.Printf("Error listing backups: %v\n", err)
		os.Exit(1)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return
	}

	// Output backups
	switch format {
	case "json":
		jsonBytes, err := json.MarshalIndent(backups, "", "  ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		fmt.Printf("Found %d backups:\n\n", len(backups))
		for i, backup := range backups {
			fmt.Printf("Backup #%d:\n", i+1)
			fmt.Printf("  Timestamp: %s\n", backup.Timestamp.Format(time.RFC3339))
			fmt.Printf("  Version: %s\n", backup.Version)
			fmt.Printf("  Description: %s\n", backup.Description)
			fmt.Printf("  File path: %s\n\n", backup.FilePath)
		}
	default:
		fmt.Printf("Unknown output format: %s\n", format)
		os.Exit(1)
	}
}

func handleRestoreCommand(path string) {
	if path == "" {
		fmt.Println("Error: backup path is required")
		os.Exit(1)
	}

	// Restore from backup
	knowledgeGraph := kg.DefaultKnowledgeGraph()
	if err := knowledgeGraph.RestoreFromBackup(path); err != nil {
		fmt.Printf("Error restoring from backup: %v\n", err)
		os.Exit(1)
	}

	// Save the restored knowledge graph
	if err := knowledgeGraph.Save("amadeus/knowledge_graph.json"); err != nil {
		fmt.Printf("Error saving restored knowledge graph: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully restored knowledge graph from backup: %s\n", path)
}
