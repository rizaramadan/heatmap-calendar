// Package testenv provides ephemeral test infrastructure using testcontainers.
package testenv

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Service represents a running instance of the application server.
type Service struct {
	// URL is the base URL of the running service.
	URL string

	// Port is the port the service is listening on.
	Port int

	// Process is the underlying OS process.
	Process *os.Process

	// cmd is the exec.Cmd (kept for cleanup).
	cmd *exec.Cmd
}

// ServiceConfig holds configuration for starting the service.
type ServiceConfig struct {
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string

	// APIKey is the API key for protected endpoints.
	APIKey string

	// SessionSecret is the session encryption secret.
	SessionSecret string

	// Port is the port to listen on (0 for random available port).
	Port int

	// BinaryPath is the path to the compiled server binary.
	// If empty, the service will be built automatically.
	BinaryPath string

	// WorkingDir is the working directory for the service.
	// Defaults to project root (for template access).
	WorkingDir string
}

// DefaultServiceConfig returns default service configuration.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		APIKey:        "test-api-key",
		SessionSecret: "test-session-secret-32-bytes!!",
		Port:          0, // Random port
	}
}

// StartService starts the Go service as a subprocess.
//
// The service is started with the provided database URL and configuration.
// It waits for the service to become ready before returning.
//
// Returns the service wrapper and cleanup function.
// Always call cleanup when done:
//
//	svc, cleanup, err := StartService(ctx, cfg)
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer cleanup()
func StartService(ctx context.Context, cfg ServiceConfig) (*Service, func(), error) {
	// Find available port if not specified
	port := cfg.Port
	if port == 0 {
		var err error
		port, err = findAvailablePort()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find available port: %w", err)
		}
	}

	// Determine working directory (project root)
	workDir := cfg.WorkingDir
	if workDir == "" {
		// Find project root by looking for go.mod
		var err error
		workDir, err = findProjectRoot()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find project root: %w", err)
		}
	}

	// Build or use provided binary
	binaryPath := cfg.BinaryPath
	if binaryPath == "" {
		var err error
		binaryPath, err = buildService(ctx, workDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build service: %w", err)
		}
	}

	// Prepare environment
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("DATABASE_URL=%s", cfg.DatabaseURL),
		fmt.Sprintf("API_KEY=%s", cfg.APIKey),
		fmt.Sprintf("SESSION_SECRET=%s", cfg.SessionSecret),
		fmt.Sprintf("PORT=%d", port),
		"LARK_APP_ID=",       // Disable Lark in tests
		"LARK_APP_SECRET=",   // Disable Lark in tests
	)

	// Start the process
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = os.Stdout // Optionally capture or discard
	cmd.Stderr = os.Stderr

	// Set process group for clean termination
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start service: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d", port)

	svc := &Service{
		URL:     url,
		Port:    port,
		Process: cmd.Process,
		cmd:     cmd,
	}

	// Wait for service to be ready
	if err := waitForService(ctx, url, 30*time.Second); err != nil {
		// Kill the process if it didn't become ready
		_ = svc.Stop()
		return nil, nil, fmt.Errorf("service failed to become ready: %w", err)
	}

	cleanup := func() {
		_ = svc.Stop()
	}

	return svc, cleanup, nil
}

// Stop gracefully stops the service.
func (s *Service) Stop() error {
	if s.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if err := s.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		_ = s.Process.Kill()
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := s.Process.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown takes too long
		_ = s.Process.Kill()
		return nil
	}
}

// HealthCheck verifies the service is responding.
func (s *Service) HealthCheck(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(s.URL + "/api/entities")
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("service returned status %d", resp.StatusCode)
	}
	return nil
}

// findAvailablePort finds a random available TCP port.
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// findProjectRoot finds the project root by looking for go.mod.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// buildService compiles the service binary.
func buildService(ctx context.Context, workDir string) (string, error) {
	binaryPath := filepath.Join(workDir, "tmp", "e2e-server")

	// Ensure tmp directory exists
	if err := os.MkdirAll(filepath.Join(workDir, "tmp"), 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp dir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/server")
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to build: %w\nOutput: %s", err, output)
	}

	return binaryPath, nil
}

// waitForService polls the service until it responds or times out.
func waitForService(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(url + "/api/entities")
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("service did not respond within %v", timeout)
}
