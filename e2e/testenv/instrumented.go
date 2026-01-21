// Package testenv provides ephemeral test infrastructure using testcontainers.
package testenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// InstrumentedService represents a running instance of the coverage-instrumented server.
//
// When the service shuts down gracefully, coverage data is automatically written
// to the GOCOVERDIR directory. Use this for E2E tests that need to capture
// code coverage from the running service.
type InstrumentedService struct {
	*Service

	// CoverageDir is the directory where coverage data is written on shutdown.
	CoverageDir string

	// BinaryPath is the path to the instrumented binary.
	BinaryPath string
}

// InstrumentedConfig holds configuration for the instrumented service.
type InstrumentedConfig struct {
	ServiceConfig

	// CoverageDir is the directory for coverage data output.
	// Defaults to coverage/e2e-service in project root.
	CoverageDir string

	// RebuildBinary forces rebuilding even if binary exists.
	RebuildBinary bool
}

// DefaultInstrumentedConfig returns default instrumented service configuration.
func DefaultInstrumentedConfig() InstrumentedConfig {
	return InstrumentedConfig{
		ServiceConfig: DefaultServiceConfig(),
		RebuildBinary: false,
	}
}

// StartInstrumentedService starts a coverage-instrumented service subprocess.
//
// The service is built with `go build -cover` to enable coverage instrumentation.
// When the service shuts down (via Stop()), coverage data is written to CoverageDir.
//
// Usage:
//
//	svc, cleanup, err := StartInstrumentedService(ctx, cfg)
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer cleanup()
//
//	// ... run tests ...
//
//	// Coverage data is written to svc.CoverageDir on cleanup
//
// After tests complete, convert coverage data with:
//
//	go tool covdata textfmt -i=coverage/e2e-service -o=coverage/e2e-service.out
func StartInstrumentedService(ctx context.Context, cfg InstrumentedConfig) (*InstrumentedService, func(), error) {
	// Find project root
	workDir := cfg.WorkingDir
	if workDir == "" {
		var err error
		workDir, err = findProjectRoot()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find project root: %w", err)
		}
	}

	// Set up coverage directory
	coverageDir := cfg.CoverageDir
	if coverageDir == "" {
		coverageDir = filepath.Join(workDir, "coverage", "e2e-service")
	}

	// Ensure coverage directory exists and is empty
	if err := os.RemoveAll(coverageDir); err != nil {
		return nil, nil, fmt.Errorf("failed to clean coverage dir: %w", err)
	}
	if err := os.MkdirAll(coverageDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create coverage dir: %w", err)
	}

	// Build instrumented binary
	binaryPath := cfg.BinaryPath
	if binaryPath == "" || cfg.RebuildBinary {
		var err error
		binaryPath, err = buildInstrumentedService(ctx, workDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build instrumented service: %w", err)
		}
	}

	// Find available port
	port := cfg.Port
	if port == 0 {
		var err error
		port, err = findAvailablePort()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find available port: %w", err)
		}
	}

	// Prepare environment with GOCOVERDIR
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("GOCOVERDIR=%s", coverageDir),
		fmt.Sprintf("DATABASE_URL=%s", cfg.DatabaseURL),
		fmt.Sprintf("API_KEY=%s", cfg.APIKey),
		fmt.Sprintf("SESSION_SECRET=%s", cfg.SessionSecret),
		fmt.Sprintf("PORT=%d", port),
		"LARK_APP_ID=",
		"LARK_APP_SECRET=",
	)

	// Start the instrumented process
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start instrumented service: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d", port)

	svc := &InstrumentedService{
		Service: &Service{
			URL:     url,
			Port:    port,
			Process: cmd.Process,
			cmd:     cmd,
		},
		CoverageDir: coverageDir,
		BinaryPath:  binaryPath,
	}

	// Wait for service to be ready
	if err := waitForService(ctx, url, 30*time.Second); err != nil {
		_ = svc.Stop()
		return nil, nil, fmt.Errorf("instrumented service failed to become ready: %w", err)
	}

	cleanup := func() {
		_ = svc.Stop()
	}

	return svc, cleanup, nil
}

// Stop gracefully stops the instrumented service, writing coverage data.
//
// Coverage data is written to CoverageDir when the process exits.
// Use SIGTERM for graceful shutdown to ensure coverage is written.
func (s *InstrumentedService) Stop() error {
	if s.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown (writes coverage)
	if err := s.Process.Signal(syscall.SIGTERM); err != nil {
		// SIGKILL won't write coverage data, but may be necessary
		_ = s.Process.Kill()
		return fmt.Errorf("failed to send SIGTERM, coverage may be incomplete: %w", err)
	}

	// Wait for process to exit and write coverage
	done := make(chan error, 1)
	go func() {
		_, err := s.Process.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		// Force kill - coverage data may be incomplete
		_ = s.Process.Kill()
		return fmt.Errorf("service shutdown timeout, coverage may be incomplete")
	}
}

// CoverageFiles returns the list of coverage data files in CoverageDir.
func (s *InstrumentedService) CoverageFiles() ([]string, error) {
	entries, err := os.ReadDir(s.CoverageDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(s.CoverageDir, entry.Name()))
		}
	}
	return files, nil
}

// HasCoverageData returns true if coverage data files exist.
func (s *InstrumentedService) HasCoverageData() bool {
	files, err := s.CoverageFiles()
	return err == nil && len(files) > 0
}

// buildInstrumentedService compiles the service with coverage instrumentation.
func buildInstrumentedService(ctx context.Context, workDir string) (string, error) {
	binaryPath := filepath.Join(workDir, "tmp", "e2e-server-instrumented")

	// Ensure tmp directory exists
	if err := os.MkdirAll(filepath.Join(workDir, "tmp"), 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp dir: %w", err)
	}

	// Build with coverage instrumentation
	// -cover enables coverage instrumentation in the binary
	cmd := exec.CommandContext(ctx, "go", "build", "-cover", "-o", binaryPath, "./cmd/server")
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to build instrumented binary: %w\nOutput: %s", err, output)
	}

	return binaryPath, nil
}

// ConvertCoverageData converts binary coverage data to text format.
//
// This runs: go tool covdata textfmt -i=<coverageDir> -o=<outputFile>
//
// Call this after the instrumented service has stopped:
//
//	err := svc.ConvertCoverageData("coverage/e2e-service.out")
func (s *InstrumentedService) ConvertCoverageData(outputFile string) error {
	if !s.HasCoverageData() {
		return fmt.Errorf("no coverage data found in %s", s.CoverageDir)
	}

	cmd := exec.Command("go", "tool", "covdata", "textfmt",
		"-i="+s.CoverageDir,
		"-o="+outputFile,
	)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to convert coverage data: %w\nOutput: %s", err, output)
	}

	return nil
}
