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
	log.Fatal(serveAddress(":"+strconv.Itoa(port), handler))
}

// startSweeper launches the alert evaluation worker bound to a cancellable
// context; the returned stop func must be called during shutdown so the
// worker goroutine does not outlive the process.
func startSweeper(sweeper *alertSweeper) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go sweeper.Run(ctx)
	return cancel
}
