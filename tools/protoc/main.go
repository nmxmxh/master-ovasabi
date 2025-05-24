package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var (
	// Regex for validating proto file names.
	protoFileRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*\.proto$`)
	// Regex for validating directory paths.
	dirPathRegex = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)
)

func validateInputs(protoFile, protoDir, outDir string) error {
	// Validate proto file name
	if !protoFileRegex.MatchString(filepath.Base(protoFile)) {
		return fmt.Errorf("invalid proto file name: %s", protoFile)
	}

	// Validate directory paths
	if !dirPathRegex.MatchString(protoDir) || !dirPathRegex.MatchString(outDir) {
		return fmt.Errorf("invalid directory path")
	}

	// Check if directories exist
	if _, err := os.Stat(protoDir); os.IsNotExist(err) {
		return fmt.Errorf("proto directory does not exist: %s", protoDir)
	}
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", outDir)
	}

	// Ensure proto file exists
	protoPath := filepath.Join(protoDir, protoFile)
	if _, err := os.Stat(protoPath); os.IsNotExist(err) {
		return fmt.Errorf("proto file does not exist: %s", protoPath)
	}

	return nil
}

func runProtoc(protoFile, protoDir, outDir string, openAPI bool) error {
	// Validate inputs
	if err := validateInputs(protoFile, protoDir, outDir); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Construct command with absolute paths
	protoPath, err := filepath.Abs(filepath.Join(protoDir, protoFile))
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	outPath, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Construct protoc command with sanitized inputs
	args := []string{
		"--go_out=" + outPath,
		"--go_opt=paths=source_relative",
		"--go-grpc_out=" + outPath,
		"--go-grpc_opt=paths=source_relative",
		"--grpc-gateway_out=" + outPath,
		"--grpc-gateway_opt=paths=source_relative",
		"-I=" + filepath.Dir(protoPath),
		protoPath,
	}

	if openAPI {
		args = append([]string{
			"--openapiv2_out=" + outPath,
			"--openapiv2_opt=logtostderr=true",
		}, args...)
	}

	// Create command with clean environment
	cmd := exec.Command("protoc", args...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("protoc failed: %w\nOutput: %s", err, output)
	}

	if openAPI {
		fmt.Printf("OpenAPI schema generated in %s\n", outPath)
	}
	return nil
}

func main() {
	if len(os.Args) < 4 || len(os.Args) > 5 {
		fmt.Printf("Usage: %s <proto_file> <proto_dir> <out_dir> [--openapi]\n", os.Args[0])
		os.Exit(1)
	}

	protoFile := os.Args[1]
	protoDir := os.Args[2]
	outDir := os.Args[3]
	openAPI := len(os.Args) == 5 && os.Args[4] == "--openapi"

	if err := runProtoc(protoFile, protoDir, outDir, openAPI); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// Usage:
//   go run main.go <proto_file> <proto_dir> <out_dir> [--openapi]
//
// If --openapi is provided, generates OpenAPI (Swagger) schema using protoc-gen-openapiv2.
// Example:
//   go run main.go metadata.proto api/protos/common/v1 docs/generated/openapi --openapi
