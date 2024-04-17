package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"

	"github.com/swagftw/rex/types"
)

var daemonAddr = "0.0.0.0:8080"

func main() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic: while running client", "err", r)
		}
	}()

	conn, err := net.Dial("tcp", daemonAddr)
	if err != nil {
		slog.Error("failed to dial to the daemon", "err", err)

		return
	}

	slog.Info("connected to the daemon")

	for {
		// check if the connection is still alive
		_, err = conn.Write([]byte{'\n'})
		if err != nil {
			slog.Error("connection is not alive", "err", err)

			return
		}

		reader := bufio.NewReader(conn)

		err = readConnection(conn, reader)
		if errors.Is(err, io.EOF) {
			return
		}
	}
}

func readConnection(conn net.Conn, reader *bufio.Reader) error {
	byteData, err := reader.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			slog.Error("connection closed", "err", err)

			return err
		}

		slog.Error("failed to read data", "err", err)

		return err
	}

	data := new(types.Data)
	err = json.Unmarshal(byteData, data)
	if err != nil {
		slog.Error("failed to unmarshal data", "err", err)

		return nil
	}

	if data.Type != types.HeartbeatPing {
		return nil
	}

	slog.Info("ping received")

	data = &types.Data{

		Type: types.HeartbeatPong,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal data", "err", err)

		return nil
	}

	buf := new(bytes.Buffer)

	err = json.Compact(buf, jsonData)
	if err != nil {
		slog.Error("failed to compact data", "err", err)

		return nil
	}

	buf.WriteByte('\n')

	_, err = conn.Write(buf.Bytes())
	if err != nil {
		slog.Error("failed to write data", "err", err)

		return nil
	}

	slog.Info("pong sent")

	return nil
}
