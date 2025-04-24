package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	protoDir = flag.String("proto-dir", "api/protos", "directory containing .proto files")
	outDir   = flag.String("out-dir", "api/protos", "output directory for generated files")
	lang     = flag.String("lang", "go", "target language (go, python, java)")
)

func main() {
	flag.Parse()

	// Find all .proto files
	protoFiles, err := findProtoFiles(*protoDir)
	if err != nil {
		fmt.Printf("Error finding proto files: %v\n", err)
		os.Exit(1)
	}

	// Generate code for each proto file
	for _, protoFile := range protoFiles {
		if err := generateCode(protoFile); err != nil {
			fmt.Printf("Error generating code for %s: %v\n", protoFile, err)
			os.Exit(1)
		}
	}
}

func findProtoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func generateCode(protoFile string) error {
	var cmd *exec.Cmd

	switch *lang {
	case "go":
		cmd = exec.Command("protoc",
			"--go_out="+*outDir,
			"--go_opt=paths=source_relative",
			"--go-grpc_out="+*outDir,
			"--go-grpc_opt=paths=source_relative",
			"-I"+*protoDir,
			protoFile,
		)
	case "python":
		cmd = exec.Command("protoc",
			"--python_out="+*outDir,
			"--grpc_python_out="+*outDir,
			"-I"+*protoDir,
			protoFile,
		)
	case "java":
		cmd = exec.Command("protoc",
			"--java_out="+*outDir,
			"--grpc-java_out="+*outDir,
			"-I"+*protoDir,
			protoFile,
		)
	default:
		return fmt.Errorf("unsupported language: %s", *lang)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
