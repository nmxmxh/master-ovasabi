package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/tester"
)

func main() {
	// Example: create a dummy suite for demonstration. In practice, import your real suite definitions.
	scenarios := []*metadata.TestScenario{
		{
			Name:        "EmitEchoEvents_Batch1000",
			Description: "Emit 1000 echo events",
			Params:      map[string]interface{}{"N": 1000},
			InitialMeta: tester.DefaultTestMeta(),
		},
	}
	suite := &metadata.TestSuite{
		Name:      "NexusBenchmarks",
		Scenarios: scenarios,
	}

	docs := tester.GenerateTestDocs(suite)
	outputDir := filepath.Join("docs", "generated", "tests")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		panic(err)
	}
	outputFile := filepath.Join(outputDir, "nexus_benchmarks.md")
	if err := writeDocs(outputFile, docs); err != nil {
		panic(err)
	}
	fmt.Printf("Test documentation generated: %s\n", outputFile)
}

// writeDocs writes the generated documentation to a file.
func writeDocs(outputFile, docs string) error {
	if err := os.WriteFile(outputFile, []byte(docs), 0o600); err != nil {
		return fmt.Errorf("failed to write docs: %w", err)
	}
	return nil
}
