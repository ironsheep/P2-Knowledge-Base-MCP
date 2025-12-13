package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetCacheDirWithEnvVar(t *testing.T) {
	// Use a temp directory that exists and is writable
	tmpDir := t.TempDir()
	os.Setenv("P2KB_CACHE_DIR", tmpDir)
	defer os.Unsetenv("P2KB_CACHE_DIR")

	dir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error: %v", err)
	}
	if dir != tmpDir {
		t.Errorf("GetCacheDir() = %q, want %q", dir, tmpDir)
	}
}

func TestGetCacheDirOrDefault(t *testing.T) {
	// This should always return a value, never panic
	dir := GetCacheDirOrDefault()
	if dir == "" {
		t.Error("GetCacheDirOrDefault() returned empty string")
	}
}

func TestEnsureWritableDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating a new directory
	newDir := filepath.Join(tmpDir, "test", "nested")
	err := ensureWritableDir(newDir)
	if err != nil {
		t.Errorf("ensureWritableDir() failed for new dir: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("directory was not created")
	}

	// Test with existing directory
	err = ensureWritableDir(newDir)
	if err != nil {
		t.Errorf("ensureWritableDir() failed for existing dir: %v", err)
	}
}

func TestIsContainerToolsInstall(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/opt/container-tools/p2kb-mcp/bin/p2kb-mcp", true},
		{"/opt/container-tools/bin/p2kb-mcp", true},
		{"/usr/local/bin/p2kb-mcp", false},
		{"/home/user/p2kb-mcp/bin/p2kb-mcp", false},
		{"/opt/other-tools/bin/p2kb-mcp", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isContainerToolsInstall(tt.path)
			if result != tt.expected {
				t.Errorf("isContainerToolsInstall(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetWindowsCacheDir(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// Test with LOCALAPPDATA set
	tmpDir := t.TempDir()
	os.Setenv("LOCALAPPDATA", tmpDir)
	defer os.Unsetenv("LOCALAPPDATA")

	dir, err := getWindowsCacheDir()
	if err != nil {
		t.Fatalf("getWindowsCacheDir() error: %v", err)
	}

	expected := filepath.Join(tmpDir, AppName, "cache")
	if dir != expected {
		t.Errorf("getWindowsCacheDir() = %q, want %q", dir, expected)
	}
}

func TestAppName(t *testing.T) {
	if AppName != "p2kb-mcp" {
		t.Errorf("AppName = %q, want p2kb-mcp", AppName)
	}
}
