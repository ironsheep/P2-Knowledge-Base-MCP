package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ironsheep/p2kb-mcp/internal/server"
)

// Version information - set by ldflags during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Handle --version and -v flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("p2kb-mcp %s\n", Version)
			fmt.Printf("  Build time: %s\n", BuildTime)
			fmt.Printf("  Git commit: %s\n", GitCommit)
			return
		case "--help", "-h", "help":
			fmt.Println("p2kb-mcp - MCP server for P2 Knowledge Base access")
			fmt.Println()
			fmt.Println("Usage: p2kb-mcp [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --version, -v    Print version information")
			fmt.Println("  --help, -h       Print this help message")
			fmt.Println()
			fmt.Println("Environment variables:")
			fmt.Println("  P2KB_CACHE_DIR     Cache directory (default: ~/.p2kb-mcp)")
			fmt.Println("  P2KB_INDEX_TTL     Index TTL in seconds (default: 86400)")
			fmt.Println("  P2KB_LOG_LEVEL     Log level: debug, info, warn, error (default: info)")
			fmt.Println()
			fmt.Println("This server communicates via MCP protocol over stdin/stdout.")
			fmt.Println("Configure it in your MCP client (e.g., Claude Desktop).")
			return
		}
	}

	// Configure logging to stderr (stdout is for MCP protocol)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	logLevel := os.Getenv("P2KB_LOG_LEVEL")
	if logLevel == "debug" {
		log.Printf("P2KB MCP Server v%s (built %s, commit %s)", Version, BuildTime, GitCommit)
	}

	srv := server.New(Version)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
