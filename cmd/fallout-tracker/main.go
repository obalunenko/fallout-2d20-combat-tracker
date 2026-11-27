package main

import (
	"context"
	"log"
	"os"
	"runtime/debug"

	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/mcpserver"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/obalunenko/fallout/internal/ui/fyneui"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// The stdio bridge for Claude Desktop runs alone: no database, no UI.
	if len(os.Args) > 1 && os.Args[1] == "mcp-bridge" {
		os.Exit(mcpserver.RunBridge(context.Background(), os.Stdin, os.Stdout, os.Stderr, os.Getenv))
	}

	lifecycleCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPath, err := sqlite.ResolveDBPath()
	if err != nil {
		log.Fatalf("failed to resolve db path: %v", err)
	}

	db, err := sqlite.OpenAndMigrate(dbPath)
	if err != nil {
		log.Fatalf("failed to init sqlite storage: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("failed to close sqlite db: %v", cerr)
		}
	}()

	repo := sqlite.NewEncounterStore(db)
	svc := app.NewService(repo)
	router := app.NewConfirmationRouter()
	conn := mcpserver.NewConnection(svc, router, mcpserver.Options{Version: resolveVersion(), Logf: log.Printf})

	if err := fyneui.Run(lifecycleCtx, svc, cancel, fyneui.Options{Assistant: conn, Confirmations: router}); err != nil {
		log.Fatalf("application failed: %v", err)
	}
}

// resolveVersion returns the ldflags version, or the module version when the
// binary was built with go install, or "dev".
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
