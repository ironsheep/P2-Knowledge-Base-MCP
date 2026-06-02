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

// TestIsContainerToolsInstall verifies the STRUCTURAL detection rule:
// a container-tools install has the binary's immediate parent directory named "platforms".
// The old substring-match on "container-tools/" is intentionally gone.
func TestIsContainerToolsInstall(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Container-tools layout: <root>/bin/platforms/<binary>
		{
			name:     "container-tools versioned binary under platforms",
			path:     filepath.Join("/opt", "ct", "p2kb-mcp", "bin", "platforms", "p2kb-mcp-v1.5.0"),
			expected: true,
		},
		{
			name:     "container-tools plain binary name under platforms",
			path:     filepath.Join("/opt", "container-tools", "bin", "platforms", "p2kb-mcp"),
			expected: true,
		},
		// Standalone layout: <root>/bin/<binary> — no "platforms" parent
		{
			name:     "standalone: binary directly under bin (old container-tools path style, no platforms)",
			path:     filepath.Join("/opt", "container-tools", "p2kb-mcp", "bin", "p2kb-mcp"),
			expected: false,
		},
		{
			name:     "standalone: old container-tools/bin path without platforms level",
			path:     filepath.Join("/opt", "container-tools", "bin", "p2kb-mcp"),
			expected: false,
		},
		{
			name:     "standalone: /usr/local/bin",
			path:     filepath.Join("/usr", "local", "bin", "p2kb-mcp"),
			expected: false,
		},
		{
			name:     "standalone: home install",
			path:     filepath.Join("/home", "user", "p2kb-mcp", "bin", "p2kb-mcp"),
			expected: false,
		},
		{
			name:     "standalone: other-tools path",
			path:     filepath.Join("/opt", "other-tools", "bin", "p2kb-mcp"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isContainerToolsInstall(tt.path)
			if result != tt.expected {
				t.Errorf("isContainerToolsInstall(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestCacheDirForExe verifies the pure layout helper covering the main install patterns.
func TestCacheDirForExe(t *testing.T) {
	tests := []struct {
		name      string
		exePath   string
		wantDir   string
		wantCt    bool // for documentation — cacheDirForExe uses isContainerToolsInstall internally
	}{
		{
			name:    "container-tools: versioned binary under bin/platforms",
			exePath: filepath.Join("/opt", "ct", "p2kb-mcp", "bin", "platforms", "p2kb-mcp-v1.5.0"),
			wantDir: filepath.Join("/opt", "ct", "p2kb-mcp", "var", "cache", AppName),
			wantCt:  true,
		},
		{
			name:    "standalone: binary directly under bin",
			exePath: filepath.Join("/opt", "app", "bin", "p2kb-mcp"),
			wantDir: filepath.Join("/opt", "app", ".cache"),
			wantCt:  false,
		},
		{
			name:    "standalone: /usr/local/bin — bin found by walk-up",
			exePath: filepath.Join("/usr", "local", "bin", "p2kb-mcp"),
			wantDir: filepath.Join("/usr", "local", ".cache"),
			wantCt:  false,
		},
		{
			name:    "container-tools: deeply nested root, platforms present",
			exePath: filepath.Join("/srv", "deploy", "v2", "release", "bin", "platforms", "p2kb-mcp-linux-arm64"),
			wantDir: filepath.Join("/srv", "deploy", "v2", "release", "var", "cache", AppName),
			wantCt:  true,
		},
		{
			name:    "standalone: home directory install",
			exePath: filepath.Join("/home", "user", "tools", "bin", "p2kb-mcp"),
			wantDir: filepath.Join("/home", "user", "tools", ".cache"),
			wantCt:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cacheDirForExe(tt.exePath)
			if got != tt.wantDir {
				t.Errorf("cacheDirForExe(%q)\n  got  %q\n  want %q", tt.exePath, got, tt.wantDir)
			}
			// Cross-check with isContainerToolsInstall for documentation
			ct := isContainerToolsInstall(tt.exePath)
			if ct != tt.wantCt {
				t.Errorf("isContainerToolsInstall(%q) = %v, want %v", tt.exePath, ct, tt.wantCt)
			}
		})
	}
}

// TestFindInstallRoot validates the walk-up-to-bin helper in isolation.
func TestFindInstallRoot(t *testing.T) {
	tests := []struct {
		name    string
		exeDir  string
		wantRoot string
	}{
		{
			name:     "exeDir is bin itself",
			exeDir:   filepath.Join("/opt", "app", "bin"),
			wantRoot: filepath.Join("/opt", "app"),
		},
		{
			name:     "exeDir is bin/platforms (container-tools)",
			exeDir:   filepath.Join("/opt", "ct", "p2kb-mcp", "bin", "platforms"),
			wantRoot: filepath.Join("/opt", "ct", "p2kb-mcp"),
		},
		{
			name:     "usr local bin",
			exeDir:   filepath.Join("/usr", "local", "bin"),
			wantRoot: filepath.Join("/usr", "local"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findInstallRoot(tt.exeDir)
			if got != tt.wantRoot {
				t.Errorf("findInstallRoot(%q)\n  got  %q\n  want %q", tt.exeDir, got, tt.wantRoot)
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
