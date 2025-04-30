package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/nmxmxh/master-ovasabi/tools/docgen"
)

func main() {
	// Parse command line flags
	sourceDir := flag.String("source", ".", "Source directory to generate documentation from")
	outputDir := flag.String("output", "docs/generated", "Output directory for generated documentation")
	flag.Parse()

	// Create absolute paths
	srcAbs, err := filepath.Abs(*sourceDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for source directory: %v", err)
	}

	outAbs, err := filepath.Abs(*outputDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for output directory: %v", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outAbs, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Create documentation generator
	generator := docgen.NewDocGenerator(srcAbs, outAbs)

	// Generate documentation
	if err := generator.Generate(); err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}

	log.Printf("Documentation generated successfully in %s", outAbs)
}
