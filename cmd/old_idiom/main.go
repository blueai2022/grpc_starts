package main

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

func main() {
	// main() becomes just an error handler - lob-sided and lacking elegant: just for log.Fatal()
	// Most importantly, remain restrictive: paralell shutdown still unsupported
	if err := run(); err != nil {
		log.Fatal().
			Err(err).
			Msg("application failed")
	}
}

func run() error {
	// All the logic originally belonged to main() got moved here
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Config

	// Resource Connections: e.g. NATS/Kafka, Cassandra and etc.
	// natsConn, err := ...
	// if err != nil {
	//    .....
	// }
	// defer natsConn.Close() // defer Close can be done here, log.Fatal is not in run() 

	// Business logic initialization, e.g. video stream controller/manager
	streamController, err := stream.NewController()
	if err != nil {
		return fmt.Errorf("failed to create new stream controller: %w", err) // error for main() to log.Fatal
	}

	// Prometheus Metrics registry setup

	// Connect RPC Service
	svc, err := service.New(
		service.WithStreamController(streamController) // Business logic dependenies injected here
	)
	if err != nil {
		return fmt.Errorf("failed to create call control service: %w", err)
	}

	// HTTP handlers setup healthHandler, metricsHandler := ...
	mux := http.NewServeMux()

	// mux.Handle("/metrics", metricsHandler)
	// mux.HandleFunc("/health", healthHandlerFunc)
	// mux.HandleFunc("/readyz", healthHandlerFunc)
	// mux.HandleFunc("/livez", versionHandlerFunc)

	// Connect Handlers and Transcoders
	// mux.Handle(svc.ConnectHandler())

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	httpServer := &http.Server{
		Addr:     cfg.HTTP.Address(),
		Handler:  mux,
		ErrorLog: stdlog.New(io.Discard, "", 0),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Protocols: protocols,
	}

	// Start HTTP server
	go func() {
		log.Info().
			Str("event_action", "start").
			Str("server_address", httpServer.Addr).
			Msg("starting http server")

		if err := httpServer.ListenAndServe(); err != nil {
			log.Error().
				Err(err).
				Msg("http server shutdown error")
		}
	}()

	defer cancel()

	<-ctx.Done()

	// shutdown logic
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().
			Err(err).
			Msg("http server shutdown error")
	}

	// Note: resource cleanup logic here, as they were handled in defer code above

	log.Warn().
		Str("event_action", "shutdown").
		Msg("shut down")

	return nil
}
