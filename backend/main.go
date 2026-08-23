package main

import (
	"context"
	"embed"
	"example.com/grain-silo-safety-service/api"
	"example.com/grain-silo-safety-service/config"
	"example.com/grain-silo-safety-service/store"
	"io/fs"
	"log"
	"strconv"
	"time"
)

//go:embed web/*
var webFiles embed.FS

func main() {
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	silos := store.New()
	opsSvc := newOpsService(opsSeedRecords())
	inspectionSvc := newInspectionService(silos, config.MaxInspectionWorkers())
	alertStore := newAlertStore()
	alertSvc := newAlertService(alertStore, silos, newOpsClock())
	sweeper := newAlertSweeper(alertSvc, 30*time.Second)

	stopSweeper := startSweeper(sweeper)
	defer stopSweeper()

	handler := api.NewRouter(silos, webFS, newOpsAPI(opsSvc), newInspectionAPI(inspectionSvc), newAlertsAPI(alertSvc))
	port := config.Port()
	log.Printf("grain silo service listening on :%d", port)
	if err := serveAddress(":"+strconv.Itoa(port), handler, stopSweeper); err != nil {
		// Start-up failure (e.g. port in use) — there is nothing graceful to
		// wait for, so abort. The normal shutdown path returns nil here and
		// falls through to the deferred stopSweeper instead.
		log.Fatal(err)
	}
	// Graceful shutdown completed; the deferred stopSweeper is the last call
	// to run so the background sweep goroutine is guaranteed to have exited
	// before main returns.
}

// startSweeper launches the alert evaluation worker bound to a cancellable
// context; the returned stop func stops the worker and blocks until it has
// exited, so the goroutine does not outlive the process.
func startSweeper(sweeper *alertSweeper) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go sweeper.Run(ctx)
	return func() {
		cancel()
		sweeper.Stop()
	}
}
