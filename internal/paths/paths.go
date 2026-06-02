// Package paths provides platform-specific path resolution for the P2KB MCP.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// AppName is the application name used in paths.
	AppName = "p2kb-mcp"
)

// GetCacheDir returns the cache directory path based on platform and installation type.
//
// Resolution order:
//  1. P2KB_CACHE_DIR environment variable (if set)
//  2. Container-tools layout: <root>/var/cache/p2kb-mcp/ (binary is at <root>/bin/platforms/<binary>)
//  3. Standalone layout: <root>/.cache/ (binary is at <root>/bin/<binary>, Linux/macOS)
//  4. Windows: %LOCALAPPDATA%\p2kb-mcp\cache\
//
// The install root is determined by walking up from the resolved executable until a
// directory named "bin" is found; the parent of that "bin" directory is the root.
// Container-tools mode is detected structurally: the binary's immediate parent must
// be named "platforms".
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

	// Platform-specific handling for Windows
	if runtime.GOOS == "windows" {
		return getWindowsCacheDir()
	}

	// Determine cache directory purely from the resolved exe path
	cacheDir := cacheDirForExe(exePath)

	if err := ensureWritableDir(cacheDir); err != nil {
		return "", fmt.Errorf("cache directory not writable at %s: %w", cacheDir, err)
	}

	return cacheDir, nil
}

// cacheDirForExe is a pure layout helper: given an already-resolved executable
// path it returns the cache directory that should be used, without touching the
// filesystem or environment variables.
//
// Algorithm:
//  1. Walk up the directory chain from the exe's parent until a directory named
//     "bin" is found.  The parent of that "bin" directory is the install root.
//  2. If the exe's immediate parent is named "platforms" (container-tools layout),
//     return <root>/var/cache/<AppName>.
//  3. Otherwise (standalone layout) return <root>/.cache.
//  4. If no "bin" directory is found while walking up (unexpected layout), fall
//     back to <exeDir>/../.cache to preserve the old standalone behavior.
func cacheDirForExe(exePath string) string {
	exeDir := filepath.Dir(exePath)

	// Detect container-tools mode: immediate parent of exe is "platforms"
	container := isContainerToolsInstall(exePath)

	// Walk up from exeDir looking for a directory named "bin"
	installRoot := findInstallRoot(exeDir)

	if container {
		// Container-tools: <root>/var/cache/<AppName>
		return filepath.Join(installRoot, "var", "cache", AppName)
	}

	// Standalone: <root>/.cache
	return filepath.Join(installRoot, ".cache")
}

// findInstallRoot walks up the directory chain from dir, looking for a directory
// whose base name is "bin".  When found it returns the parent of that directory
// (the install root).
//
// If no "bin" ancestor is found (e.g. the binary was invoked from an unexpected
// path layout) the function falls back to dir's own parent so callers remain
// robust — this mirrors the legacy standalone ".." behaviour.
func findInstallRoot(dir string) string {
	current := dir
	for {
		base := filepath.Base(current)
		if base == "bin" {
			return filepath.Dir(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			// Reached the filesystem root without finding "bin".
			// Fall back: treat exeDir itself as one level below root.
			return filepath.Dir(dir)
		}
		current = parent
	}
}

// isContainerToolsInstall reports whether exePath is a container-tools install.
// The detection is purely structural: a container-tools binary lives inside a
// directory named "platforms" (i.e. the path looks like <root>/bin/platforms/<binary>).
func isContainerToolsInstall(exePath string) bool {
	return filepath.Base(filepath.Dir(exePath)) == "platforms"
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
