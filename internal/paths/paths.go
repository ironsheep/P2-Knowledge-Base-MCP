// Package paths provides platform-specific path resolution for the P2KB MCP.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// AppName is the application name used in paths.
	AppName = "p2kb-mcp"
)

// GetCacheDir returns the cache directory path based on platform and installation type.
//
// Resolution order:
//  1. P2KB_CACHE_DIR environment variable (if set)
//  2. Container-tools layout: {exe}/../../../var/cache/p2kb-mcp/ (if in container-tools)
//  3. Standalone layout: {exe}/../.cache/ (Linux/macOS)
//  4. Windows: %LOCALAPPDATA%\p2kb-mcp\cache\
//
// On Linux/macOS, this function will fail if the cache directory cannot be created
// or is not writable, rather than silently falling back to another location.
func GetCacheDir() (string, error) {
	// Priority 1: Explicit environment variable override
	if dir := os.Getenv("P2KB_CACHE_DIR"); dir != "" {
		if err := ensureWritableDir(dir); err != nil {
			return "", fmt.Errorf("P2KB_CACHE_DIR is set but not writable: %w", err)
		}
		return dir, nil
	}

	// Get executable path for relative resolution
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to determine executable path: %w", err)
	}

	// Resolve symlinks to get the real path
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	// Priority 2: Container-tools layout detection
	// Path pattern: /opt/container-tools/p2kb-mcp/bin/p2kb-mcp
	// Cache location: /opt/container-tools/var/cache/p2kb-mcp/
	if isContainerToolsInstall(exePath) {
		// From {container-tools}/p2kb-mcp/bin/ go up to {container-tools}/var/cache/p2kb-mcp/
		containerToolsRoot := filepath.Dir(filepath.Dir(exeDir))
		cacheDir := filepath.Join(containerToolsRoot, "var", "cache", AppName)
		if err := ensureWritableDir(cacheDir); err != nil {
			return "", fmt.Errorf("container-tools cache directory not writable: %w", err)
		}
		return cacheDir, nil
	}

	// Platform-specific handling
	if runtime.GOOS == "windows" {
		return getWindowsCacheDir()
	}

	// Priority 3: Standalone layout (Linux/macOS)
	// Binary at: {install}/bin/p2kb-mcp
	// Cache at: {install}/.cache/
	cacheDir := filepath.Join(exeDir, "..", ".cache")
	cacheDir, err = filepath.Abs(cacheDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve cache path: %w", err)
	}

	if err := ensureWritableDir(cacheDir); err != nil {
		return "", fmt.Errorf("cache directory not writable at %s: %w", cacheDir, err)
	}

	return cacheDir, nil
}

// GetCacheDirOrDefault returns the cache directory, falling back to a default on error.
// This is provided for backward compatibility but logs a warning when falling back.
func GetCacheDirOrDefault() string {
	dir, err := GetCacheDir()
	if err != nil {
		// Log the error to stderr so it's visible
		fmt.Fprintf(os.Stderr, "Warning: %v; using fallback cache location\n", err)
		// Fallback to home directory (legacy behavior)
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return ".p2kb-mcp"
		}
		return filepath.Join(home, ".p2kb-mcp")
	}
	return dir
}

// getWindowsCacheDir returns the Windows-specific cache directory.
func getWindowsCacheDir() (string, error) {
	// Use LOCALAPPDATA on Windows (e.g., C:\Users\{user}\AppData\Local\p2kb-mcp\cache\)
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
	}

	cacheDir := filepath.Join(localAppData, AppName, "cache")
	if err := ensureWritableDir(cacheDir); err != nil {
		return "", fmt.Errorf("Windows cache directory not writable: %w", err)
	}

	return cacheDir, nil
}

// isContainerToolsInstall checks if the executable is running from a container-tools installation.
func isContainerToolsInstall(exePath string) bool {
	// Check if path contains "container-tools" directory
	// Normalize path separators for comparison
	normalizedPath := filepath.ToSlash(exePath)
	return strings.Contains(normalizedPath, "container-tools/")
}

// ensureWritableDir ensures a directory exists and is writable.
// Creates the directory if it doesn't exist.
func ensureWritableDir(dir string) error {
	// Try to create the directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Verify it's writable by creating a test file
	testFile := filepath.Join(dir, ".write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}
