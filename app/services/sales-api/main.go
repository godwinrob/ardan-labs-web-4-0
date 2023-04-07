package main

import (
	"github.com/godwinrob/service/foundation/logger"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var build = "develop"

func main() {

	log.Println("service starting:", build)

	// Call the logging service
	sugar, err := logger.New("SALES-API")
	if err != nil {
		log.Println("failed to created logger: " + err.Error())
		os.Exit(1)
	}
	defer sugar.Sync()

	if err := run(sugar); err != nil {
		sugar.Errorw("startup", "ERROR", err)
		if err = sugar.Sync(); err != nil {
			log.Println("failed to flush sugar logger: " + err.Error())
		}
		os.Exit(1)
	}

	defer log.Println("service stopped")
}

func run(sugar *zap.SugaredLogger) error {

	////////////////////////////////////////////////////////////////////////
	// GOMAXPROCS

	// Get max available CPUs for this machine
	sugar.Infow("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	////////////////////////////////////////////////////////////////////////
	// SIGNALS

	// Hold at shutdown until interrupt received from console
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	sugar.Infow("interrupt", "INTERUPPT", "user interrupt")
	return nil
}
