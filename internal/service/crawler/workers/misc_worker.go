//go:build linux
// +build linux

package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"github.com/sirupsen/logrus"
)

// Firecracker Execution (revised)
func runInFirecracker(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	// Setup paths
	taskID := task.Uuid
	baseDir := filepath.Join(os.TempDir(), "firecracker", taskID)
	inputFile := filepath.Join(baseDir, "task.json")
	outputFile := filepath.Join(baseDir, "result.json")
	socketPath := filepath.Join(baseDir, "fc.sock")
	kernelPath := "vmlinux"
	rootfsPath := "rootfs.ext4"

	// Prepare directories
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base dir: %w", err)
	}

	// Write input file
	inputData, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("failed to encode task: %w", err)
	}
	if err := os.WriteFile(inputFile, inputData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Configure Firecracker machine
	machineCfg := models.MachineConfiguration{
		VcpuCount:  firecracker.Int64(1),
		MemSizeMib: firecracker.Int64(128),
	}

	_ = firecracker.VMCommandBuilder{}.
		WithBin("/usr/bin/firecracker").
		WithSocketPath(socketPath).
		Build(ctx)

	cfg := firecracker.Config{
		SocketPath:      socketPath,
		LogFifo:         filepath.Join(baseDir, "fc.log"),
		MetricsFifo:     filepath.Join(baseDir, "metrics.fifo"),
		KernelImagePath: kernelPath,
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off",

		MachineCfg: machineCfg,

		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("rootfs"),
				PathOnHost:   firecracker.String(rootfsPath),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
			},
			{
				DriveID:    firecracker.String("input"),
				PathOnHost: firecracker.String(inputFile),
				IsReadOnly: firecracker.Bool(true),
			},
			{
				DriveID:    firecracker.String("output"),
				PathOnHost: firecracker.String(outputFile),
				IsReadOnly: firecracker.Bool(false),
			},
		},
	}

	vmCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Initialize a new logrus logger for Firecracker, wrapping the zap logger
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(os.Stdout) // Or redirect to a file/buffer
	logrusLogger.SetLevel(logrus.DebugLevel)

	machineOpts := []firecracker.Opt{
		firecracker.WithLogger(logrus.NewEntry(logrusLogger)),
	}

	m, err := firecracker.NewMachine(vmCtx, cfg, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	defer func() {
		_ = m.StopVMM()
	}()

	// Start the microVM
	if err := m.Start(vmCtx); err != nil {
		return nil, fmt.Errorf("failed to start firecracker VM: %w", err)
	}

	// Wait for result.json
	for i := 0; i < 40; i++ {
		if _, err := os.Stat(outputFile); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	// Read and parse result
	resultData, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	var result crawlerpb.CrawlResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}

	return &result, nil
}

// GVisor sandbox (optional)
func runInGVisor(cmd *exec.Cmd) error {
	cmd.Path = "/usr/bin/runsc"
	cmd.Args = append([]string{
		"run", "--rootless", "--network=none",
		"--fs=tmpfs:/tmp", "--watchdog-action=panic",
	}, cmd.Args...)
	return cmd.Run()
}

// Parse VM output placeholder
func parseVMMachineOutput(data []byte) *crawlerpb.CrawlResult {
	// Example: JSON unmarshal
	var result crawlerpb.CrawlResult
	_ = json.Unmarshal(data, &result)
	return &result
}

func cleanupFirecrackerArtifacts() error {
	baseDir := filepath.Join(os.TempDir(), "firecracker")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		// If the directory doesn't exist, there's nothing to clean
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read firecracker base dir: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(baseDir, entry.Name())

		// Safety check: only remove subdirs that look like task UUIDs (optional)
		if entry.IsDir() && looksLikeTaskUUID(entry.Name()) {
			if err := os.RemoveAll(entryPath); err != nil {
				// Log and continue on error, don't fail whole cleanup
				fmt.Printf("warning: failed to remove firecracker dir %s: %v\n", entryPath, err)
			}
		}
	}

	return nil
}

// Optional: crude UUID shape checker (replace with stricter regex if needed)
func looksLikeTaskUUID(name string) bool {
	return len(name) >= 8 && !strings.ContainsAny(name, " \t\n/\\")
}
