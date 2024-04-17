package heartbeat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/swagftw/rex/types"
)

var port = 8080

func StartListener() (net.Listener, error) {
	slog.Info("starting listener...", "port", port)
	listener, err := net.Listen("tcp", "0.0.0.0:"+fmt.Sprintf("%d", port))
	if err != nil {
		slog.Error("failed to start listener", "err", err)

		return nil, err
	}

	return listener, nil
}

func StartHeartbeat() error {
	listener, err := StartListener()
	if err != nil {
		return err
	}

	var conn net.Conn

	for {
		conn, err = listener.Accept()
		if err != nil {
			slog.Error("failed to accept connection", "err", err)

			continue
		}

		// write ping messages to the client
		ticker := time.NewTicker(time.Second * 2)

		go func(conn net.Conn, ticker *time.Ticker) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic: while sending ping", "err", r)
				}
			}()

			defer func() {
				err = conn.Close()
				if err != nil && !errors.Is(err, net.ErrClosed) {
					slog.Error("failed to close connection", "err", err)
				}
			}()

			for {
				if <-ticker.C; true {
					data := types.Data{
						Type: types.HeartbeatPing,
					}

					buf := &bytes.Buffer{}

					jsonData, err := json.Marshal(data)
					if err != nil {
						slog.Error("failed to marshal data", "err", err)

						panic(err)
					}

					err = json.Compact(buf, jsonData)
					if err != nil {
						slog.Error("failed to compact data", "err", err)

						panic(err)
					}

					buf.WriteByte('\n')

					_, err = conn.Write(buf.Bytes())
					if err != nil {
						ticker.Stop()

						slog.Error("failed to send ping", "err", err)

						return
					}

					slog.Info("ping sent")
				}
			}
		}(conn, ticker)

		go func(conn net.Conn) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic: while reading the heartbeat", "err", r)
				}
			}()

			defer func() {
				err = conn.Close()
				if err != nil && !errors.Is(err, net.ErrClosed) {
					slog.Error("failed to close connection", "err", err)
				}
			}()

			reader := bufio.NewReader(conn)

			for {
				var buf []byte

				buf, err = reader.ReadBytes('\n')
				if err != nil {
					slog.Error("failed to read the bytes", "err", err)

					if errors.Is(err, net.ErrClosed) {
						return
					}

					return
				}

				if len(buf) < 1 || buf[0] == '\n' {
					continue
				}

				cd := new(types.Data)

				err = json.Unmarshal(buf, cd)
				if err != nil {
					slog.Error("failed to unmarshal the bytes", "err", err)

					continue
				}

				switch cd.Type {
				case types.HeartbeatPong:
					slog.Info("pong received")
				case types.RegisterClient:
					registerClient(cd)
				case types.Close:
					return
				default:
					slog.Error("unknown type", "type", cd.Type)
				}
			}
		}(conn)
	}
}

func registerClient(cd *types.Data) {
	slog.Info("client registered", "data", cd)
}
