package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/swagftw/rex"
	"github.com/swagftw/rex/daemon/heartbeat"
	"github.com/swagftw/rex/daemon/server"
)

var port = 8081

func main() {
	errChan := make(chan error)
	sig := make(chan os.Signal, 1)

	verbose := flag.Bool("verbose", false, "verbose logging")

	flag.Parse()

	rex.InitLogger(*verbose)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		err := heartbeat.StartHeartbeat()
		if err != nil {
			errChan <- err
		}
	}()

	addr := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", port))
	listener, err := server.StartListener(addr)
	if err != nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("failed to play with connection", "err", r)
			}
		}()

		for {
			var conn net.Conn

			conn, err = listener.Accept()
			if err != nil {
				slog.Error("failed to accept connection", "err", err)

				continue
			}

			go serveReq(conn)
		}
	}()

	select {
	case err = <-errChan:
		slog.Error("failed to start daemon", "err", err)
	case <-sig:
		slog.Info("shutting down...")
	}
}

func serveReq(conn net.Conn) {
	bufSize := 1 << 6

	buf := bytes.NewBuffer(make([]byte, 0, bufSize))

	totalByteRead := 0
	for {
		tmpBuf := make([]byte, bufSize)

		n, err := conn.Read(tmpBuf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			slog.Error("failed to read data", "err", err)

			return
		}

		totalByteRead += n

		if totalByteRead >= buf.Cap() {
			buf.Grow(buf.Cap() * 2)
		}

		buf.Write(tmpBuf[:n])

		if n < bufSize {
			break
		}
	}

	for {
		line, err := buf.ReadBytes('\r')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			slog.Error("failed to read line", "err", err)
		}

		line, _ = bytes.CutPrefix(line, []byte{'\n'})
		line, _ = bytes.CutSuffix(line, []byte{'\r'})
	}

	_, err := conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	if err != nil {
		slog.Error("failed to write response", "err", err)

		return
	}

	err = conn.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		slog.Error("failed to close connection", "err", err)

		return
	}
}
