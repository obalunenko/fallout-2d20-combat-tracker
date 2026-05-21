package main

import (
	"context"
	"log"

	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/obalunenko/fallout/internal/ui/fyneui"
)

func main() {
	lifecycleCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPath, err := sqlite.DefaultDBPath()
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

	if err := fyneui.Run(lifecycleCtx, svc, cancel); err != nil {
		log.Fatalf("application failed: %v", err)
	}
}
