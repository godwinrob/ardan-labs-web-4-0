package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/godwinrob/service/foundation/logger"

	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
)

var build = "0.0"
var environment = "local"

func main() {

	log.Printf("service starting: version %s in %s environment", build, environment)

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
	opt := maxprocs.Logger(sugar.Infof)
	if _, err := maxprocs.Set(opt); err != nil {
		return fmt.Errorf("maxprocs: %w", err)
	}
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
