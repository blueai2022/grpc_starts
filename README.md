# grpc_starts

# Resource Cleanup Strategy - using a Connect gRPC example

## Problem
`log.Fatal()` calls `os.Exit()` which skips all `defer` statements, causing resource leaks on startup failures. The dilemma: we want fast-fail behavior for startup errors (exit immediately) while also ensuring proper resource cleanup. Traditional Go patterns (`defer` or `run()` function) either don't work with `log.Fatal()` or add significant complexity. We need a solution that's both production-ready and maintainable.

## Options Considered

1. **Explicit Cleanup** (original code)
   - Cleanup only at end of `main()` after successful startup
   - Startup failures exit immediately without cleanup
   - Simple but leaks resources on startup errors

2. **`run()` Function Pattern**
   - Wrap logic in `run() error` function
   - Allows proper use of `defer`
   - More idiomatic Go, but adds complexity and changes error flow

3. **`defer` + Replace `Fatal`**
   - Use `defer` after resource creation
   - Replace `log.Fatal()` with `log.Error()` + `os.Exit(1)`
   - Still has the same problem with `os.Exit()`

## Solution: Server Struct with Cleanup Method

```go
type server struct {
    settings   *config.Settings
    natsConn   *nats.Conn
    httpServer *http.Server
}

func (s *server) cleanup() {
    // Ordered shutdown: HTTP → NATS
}

func setupServer(ctx context.Context) (*server, error) {
    srv := &server{}
    // Initialize resources, return early on errors
    return srv, nil
}

func main() {
    srv, err := setupServer(ctx)
    if err != nil {
        if srv != nil {
            srv.cleanup()  // Cleans up partial state
        }

        log.Fatal().
            Err(err).
            Msg("startup failed")
    }

	go func() {
		if err := srv.httpServer.ListenAndServe(); err != nil {
			log.Error().
				Err(err).
				Msg("http server shutdown error")
		}
	}()
    
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

    srv.cleanup()  // Same cleanup on graceful shutdown
}
```

**Benefits:**
- Single cleanup path (DRY)
- Handles partial initialization
- Works with `log.Fatal()`
- Clear shutdown ordering, including parallel shutdowns
- Simple and maintainable