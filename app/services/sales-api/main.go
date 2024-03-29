package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/godwinrob/service/app/services/sales-api/handlers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/godwinrob/service/business/web/v1/debug"
	"github.com/godwinrob/service/foundation/logger"

	"github.com/ardanlabs/conf/v3"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
)

var build = "0.0"
var environment = "local"

func main() {

	// Call the logging service
	sugar, err := logger.New("SALES-API")
	if err != nil {
		log.Println("failed to created logger: " + err.Error())
		os.Exit(1)
	}
	defer sugar.Sync()

	// Run the main service
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
	// ARDANLABS CONF

	cfg := struct {
		conf.Version
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:3000"`
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			APIToken        string        `conf:"default:testFakeAPIToken,mask"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
		}
	}{
		Version: conf.Version{
			Build: build,
			Desc:  "sales-api",
		},
	}

	// Parse OS args and environment variables
	const prefix = "SALES"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			log.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// Output the configuration settings
	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	sugar.Infow("startup", "config", out)

	////////////////////////////////////////////////////////////////////////
	// START DEBUG SERVICE

	sugar.Infow("startup", "status", "debug v1 router started", "host", cfg.Web.DebugHost)

	go func() {
		if err := http.ListenAndServe(cfg.Web.DebugHost, debug.DebugStandardLibraryMux()); err != nil {
			sugar.Errorw("shutdown", "status", "debug v1 router closed with error", "host", cfg.Web.DebugHost, "ERROR", err)
		}
	}()

	////////////////////////////////////////////////////////////////////////
	// RUN THE SERVICE

	sugar.Infow("startup", "service starting: V"+build+" API support")

	// Hold at shutdown until interrupt received from console
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	apiMux := handlers.APIMux(handlers.APIMuxConfig{
		Shutdown: shutdown,
		Log:      sugar,
	})

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      apiMux,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     zap.NewStdLog(sugar.Desugar()),
	}

	serverErrors := make(chan error, 1)
	go func() {
		sugar.Infow("startup", "status", "api router started", "host", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	////////////////////////////////////////////////////////////////////////
	// SHUTDOWN

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		sugar.Infow("shutdown", "status", "shutdown started", "signal", sig)
		defer sugar.Infow("shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
