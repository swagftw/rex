package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net"

	"github.com/swagftw/rex"
)

var daemonAddr = "0.0.0.0:8080"

func main() {
	verbose := flag.Bool("verbose", false, "verbose logging")

	flag.Parse()
	rex.InitLogger(*verbose)

	conn, err := net.Dial("tcp", daemonAddr)
	if err != nil {
		slog.Error("failed to dial to the daemon", "err", err)

		return
	}

	slog.Info("connected to the daemon")

	reader := bufio.NewReader(conn)

	err = registerClient(conn)
	if err != nil {
		return
	}

	for {
		err = readConnection(reader)
		if errors.Is(err, io.EOF) {
			return
		}
	}
}

func registerClient(conn net.Conn) error {
	err := writeToConn(conn, []byte("REX REGISTER\r\n"))
	if err != nil {
		return err
	}

	slog.Info("registered client")

	return nil
}

func readConnection(reader *bufio.Reader) error {
	for {
		byteData, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Debug("connection closed", "err", err)

				return err
			}

			slog.Error("failed to read data", "err", err)

			return err
		}

		// end of the message
		if bytes.Equal(byteData, []byte{'\r', '\n', '\r', '\n'}) {
			break
		}

		// check if the message is "ping"
		if bytes.HasPrefix(byteData, []byte("REX PING")) {
			continue
		}
	}

	return nil
}

func writeToConn(conn net.Conn, data []byte) error {
	data = append(data, []byte{'\r', '\n'}...)

	_, err := conn.Write(data)
	if err != nil {
		slog.Error("failed to write data", "err", err)

		return err
	}

	return nil
}
