package main

import (
	"context"
	"crypto/rsa"
	"expvar"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/dgrijalva/jwt-go"
	"github.com/pavel418890/service/app/sales-api/handlers"
	"github.com/pavel418890/service/business/auth"
	"github.com/pavel418890/service/foundation/database"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/internal/global"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

/*
Need to figure out timeouts for http service.
You might want to reset your DB_HOST env var during test tear down.
Service should start even without a DB running yet.
*/
// build is the git version of this programm. It's set using build flags in the
var build = "develop"

func main() {
	log := log.New(os.Stdout, "SALES : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(log); err != nil {
		log.Println("main: error:", err)
		os.Exit(1)
	}

}

func run(log *log.Logger) error {
	// =======================================================================
	// Configuration

	var cfg struct {
		conf.Version
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:3000"`
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:5s"`
			ShutdownTimeout time.Duration `conf:"default:5s"`
		}
		Auth struct {
			KeyID          string `conf:"default:920ee610-06ee-4f4e-a105-8fb95be31155"`
			PrivateKeyFile string `conf:"default:/service/private.pem"`
			Algorithm      string `conf:"default:RS256"`
		}
		DB struct {
			User       string `conf:"default:postgres"`
			Password   string `conf:"default:postgres,noprint"`
			Host       string `conf:"default:db"`
			Name       string `conf:"default:postgres"`
			DisableTLS bool   `conf:"default:true"`
		}
		Zipkin struct {
			ReporterURI string  `conf:"default:zipkin:9411/api/v2/spans"`
			ServiceName string  `conf:"default:sales-api"`
			Probability float64 `conf:"default:0.05"`
		}
	}
	cfg.Version.Desc = "copyright information here"
	cfg.Version.SVN = build
	if err := conf.Parse(os.Args[1:], "SALES", &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage("SALES", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString("SALES", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config version")
			}
			fmt.Println(version)
			return nil

		}
		return errors.Wrap(err, "parsing config")
	}
	// ======================================================================
	// App Starting
	expvar.NewString("build").Set(build)
	log.Printf("main : Started : Application initialization : version %q", build)
	defer log.Println("main : Completed")

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}

	log.Printf("main : Config :\n%v\n", out)

	//=========================================================================
	// Initialize authentication support
	log.Println("main: Started : Initializing authentication support")

	privatePEM, err := os.ReadFile(cfg.Auth.PrivateKeyFile)
	if err != nil {
		return errors.Wrap(err, "reading auth private key")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return errors.Wrap(err, "parsing auth privatekey")
	}
	lookup := func(kid string) (*rsa.PublicKey, error) {
		switch kid {
		case cfg.Auth.KeyID:
			return &privateKey.PublicKey, nil
		}
		return nil, fmt.Errorf("no public key found for the specified kid: %s", kid)
	}
	auth, err := auth.New(cfg.Auth.Algorithm, lookup, auth.Keys{cfg.Auth.KeyID: privateKey})
	if err != nil {
		return errors.Wrap(err, "constructing auth")
	}
	// ========================================================================
	// Start Tracing Support

	// WARNINGS: The current Init settings are using defaults which may not be
	// compatible with your project. Please review the documentation for
	// opentelemetry.
	log.Println("main: Initializing OT/Zipkin tracing support")
	exporter, err := zipkin.New(
		cfg.Zipkin.ReporterURI,
		cfg.Zipkin.ServiceName,
		zipkin.WithLogger(log),
	)
	if err != nil {
		return errors.Wrap(err, "creating new exporter")
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.TracerProviderOption(
			sdktrace.TraceIDRatioBased(cfg.Zipkin.Probability),
			sdktrace.WithBatcher(exporter, sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize)),
			sdktrace.WithBatchTimeout(sdktrace.DefaultExportTimeout),
			sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
		),
	)
	global.SetTracerProvider(tp)
	// ========================================================================
	// Start database
	log.Println("main: Initializing database support")
	cfgDB := database.Config{
		User:       cfg.DB.User,
		Password:   cfg.DB.Password,
		Host:       cfg.DB.Host,
		Name:       cfg.DB.Name,
		DisableTLS: cfg.DB.DisableTLS,
	}
	db, err := database.Open(cfgDB)
	if err != nil {
		return errors.Wrap(err, "connecting to db")
	}
	defer func() {
		log.Printf("main: Database Stopping : %s", cfg.DB.Host)
		db.Close()
	}()

	// Start Debug Service
	//
	// debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down the the application is shutdown.
	log.Println("main : Initializing debbuging support")

	go func() {
		log.Printf("main : Debug Listening %s", cfg.Web.DebugHost)
		if err := http.ListenAndServe(cfg.Web.DebugHost, http.DefaultServeMux); err != nil {
			log.Printf("main : Debug Listener closed : %v", err)
		}
	}()

	//=========================================================================
	// Start API Service

	log.Println("main : Initializing API support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use  a buffered channel because the signal package requires it.

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      handlers.API(build, shutdown, log, auth, db),
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Printf("main : API litening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()
	// Blocking main and waiting for shutdown
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")
	case sig := <-shutdown:
		log.Printf("main : %v  : Start shutdown", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and shed load.
		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return errors.Wrap(err, "could not stop server gracefully")
		}
	}

	return nil

}
