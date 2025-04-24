package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

var (
	verbose = flag.Bool("v", false, "verbose output")
	fix     = flag.Bool("fix", false, "automatically fix issues")
)

func main() {
	flag.Parse()

	// Get the root directory of the project
	_, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Run all linters
	runLinter("golangci-lint", "run", "--config", ".golangci.yml")
	runLinter("staticcheck", "./...")
	runLinter("gosec", "./...")
	runLinter("errcheck", "./...")
	runLinter("ineffassign", "./...")
	runLinter("unused", "./...")
}

func runLinter(name string, args ...string) {
	fmt.Printf("Running %s...\n", name)

	// Add --fix flag if requested
	if *fix {
		args = append([]string{"--fix"}, args...)
	}

	cmd := exec.Command(name, args...)
	if *verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("%s found issues\n", name)
			os.Exit(exitErr.ExitCode())
		}
		fmt.Printf("Error running %s: %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("%s completed successfully\n", name)
}
