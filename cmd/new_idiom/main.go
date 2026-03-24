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

	"github.com/blueai2022/grpc_starts/internal/config"
	"github.com/blueai2022/grpc_starts/internal/stream"
	"github.com/rs/zerolog/log"
)

type server struct {
	// Config
	settings config.Settings

	// Resources such as nats.Conn
	// natsConn   *nats.Conn

	// Dependencies
	streamController stream.Controller

	// HTTP server set up along with other essential resources
	httpServer *http.Server
}

// cleanup release resources for fatal error duing startup or after httpServer shutdown
//
// Resources examples: nats, kafka, cassandra
func (s *server) cleanup() {

	// natsConn.Close()
}

func setupServer(ctx context.Context) (*server, error) {
	srv := &server{}

	// Config
	// TODO: add config.New() with err rerurned
	srv.settings = config.Settings{}

	controller, err := stream.NewController()
	if err != nil {
		return nil, fmt.Errorf("failed to create new stream controller: %w", err)
	}

	srv.streamController = controller

	// HTTP handlers setup healthHandler, metricsHandler := ...
	mux := http.NewServeMux()

	// mux.Handle("/metrics", metricsHandler)
	// mux.HandleFunc("/health", healthHandlerFunc)

	// Connect Handlers and Transcoders, to serve url path /video
	// mux.Handle(svc.ConnectHandler())

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	httpServer := &http.Server{
		// Addr:     cfg.HTTP.Address(),
		Handler:  mux,
		ErrorLog: stdlog.New(io.Discard, "", 0),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Protocols: protocols,
	}

	srv.httpServer = httpServer

	return srv, nil
}

func main() {
	// All the logic originally belonged to main() got moved here
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv, err := setupServer(ctx)
	if err != nil {
		if srv != nil {
			srv.cleanup() // Cleans up partial state
		}

		log.Fatal().
			Err(err).
			Msg("startup failed")
	}

	go func() {
		log.Info().
			Str("event_action", "start").
			Str("server_address", srv.httpServer.Addr).
			Msg("starting http server")

		if err := srv.httpServer.ListenAndServe(); err != nil {
			log.Error().
				Err(err).
				Msg("http server shutdown error")
		}
	}()

	defer cancel()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		srv.settings.HTTP.ShutdownTimeout,
	)
	defer shutdownCancel()

	if err := srv.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().
			Err(err).
			Msg("http server shutdown error")
	}

	// Choice of same cleanup on graceful shutdown
	// or paralell shutdown.
	srv.cleanup()

	log.Info().
		Str("event_action", "shutdown").
		Msg("shut down complete")
}
