package main

import (
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/godwinrob/service/app/services/sales-api/handlers"

	"github.com/ardanlabs/conf"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var build = "develop"

//var version = "0"

func main() {

	// Call zap and enable sugar logger
	sugar, err := initLog("SALES-API")
	if err != nil {
		log.Println("failed to enable zap logging. error: " + err.Error())
		os.Exit(1)
	}

	// Perform startup and run service
	if err := run(sugar); err != nil {
		sugar.Errorw("run", "ERROR", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(sugarLog *zap.SugaredLogger) error {

	// Set max threads to the max cores available from system
	if _, err := maxprocs.Set(); err != nil {
		return fmt.Errorf("maxprocs error: %s", err.Error())
	}
	sugarLog.Infow("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	////////////////////////////////////////////////////////////////////////
	// Ardan Labs configuration

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
			SVN:  build,
			Desc: "sales-api",
		},
	}

	// Parse OS args and environment variables
	const prefix = "SALES"
	help, err := conf.ParseOSArgs(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			log.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	////////////////////////////////////////////////////////////////////////
	// Run the service
	sugarLog.Infow("starting service", "version", build)
	defer sugarLog.Infow("shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output failed: %w", err)
	}

	sugarLog.Infow("startup", "config", out)
	expvar.NewString("build").Set(build)

	////////////////////////////////////////////////////////////////////////
	// Start Debug Service

	sugarLog.Infow("startup", "status", "debug router started", "host", cfg.Web.DebugHost)

	// Construct the mux for debug calls
	debugMux := handlers.DebugStandardLibraryMux()

	// Start the debug service listening for requests
	go func() {
		if err := http.ListenAndServe(cfg.Web.DebugHost, debugMux); err != nil {
			sugarLog.Errorw("shutdown", "status", "debug router closed", cfg.Web.DebugHost, "ERROR", err)
		}
	}()

	////////////////////////////////////////////////////////////////////////
	// hold at shutdown until interrupt received from console
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	log.Println("service stopped")

	return nil
}

func initLog(service string) (*zap.SugaredLogger, error) {
	// Set zap logging config
	zapConfig := zap.NewProductionConfig()
	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.DisableStacktrace = true
	zapConfig.InitialFields = map[string]interface{}{
		"service": service,
	}

	// Build logger with config
	zapLog, err := zapConfig.Build()
	if err != nil {
		return nil, errors.New("failed to build zapConfig. error: " + err.Error())
	}
	defer zapLog.Sync()

	// Enable sugar logger
	sugar := zapLog.Sugar()
	if err := run(sugar); err != nil {
		return nil, errors.New("failed to enable sugar logger. error: " + err.Error())
	}

	return sugar, nil
}
