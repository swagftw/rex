package server

import (
	"log/slog"
	"net"
)

func StartListener(addr string) (net.Listener, error) {
	slog.Info("starting listener", "addr", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to start server", "err", err)

		return nil, err
	}

	return listener, nil
}
