package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/swagftw/rex/daemon/heartbeat"
)

var port = 8081

func main() {
	errChan := make(chan error)
	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		err := heartbeat.StartHeartbeat()
		if err != nil {
			errChan <- err
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	})

	slog.Info("starting server...", "port", port)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			errChan <- err
		}

		slog.Error("failed to start server", "err", err)
	}()

	select {
	case err := <-errChan:
		slog.Error("failed to start daemon", "err", err)
	case <-sig:
		slog.Info("shutting down...")
	}
}
