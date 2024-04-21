package heartbeat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid"

	"github.com/swagftw/rex/types"
)

var port = 8080

var ActiveClients = sync.Map{}
var clientLock = sync.RWMutex{}

func StartListener() (net.Listener, error) {
	slog.Info("starting heartbeat...", "port", port)

	hostPort := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", port))

	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		slog.Error("failed to start heartbeat", "err", err)

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

		go func(conn net.Conn) {
			// write ping messages to the client
			ticker := time.NewTicker(time.Second * 2)

			defer ticker.Stop()

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
					err = PingClient(conn)
					if err != nil {
						return
					}
				}
			}
		}(conn)

		// read from connection
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
					if errors.Is(err, io.EOF) {
						return
					}

					slog.Error("failed to read the bytes", "err", err)

					return
				}

				bytes.TrimSuffix(buf, []byte{'\r'})

				cd := new(types.Data)

				err = json.Unmarshal(buf, cd)
				if err != nil {
					slog.Error("failed to unmarshal the bytes", "err", err)

					continue
				}

				switch cd.Type {
				case types.RegisterClient:
					err = registerClient(conn)
					if err != nil {
						return
					}
				case types.Close:
					return
				default:
					slog.Error("unknown type", "type", cd.Type)
				}
			}
		}(conn)
	}
}

func PingClient(conn net.Conn) error {
	data := types.Data{
		Type: types.HeartbeatPing,
	}

	buf := &bytes.Buffer{}

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal data", "err", err)

		return err
	}

	err = json.Compact(buf, jsonData)
	if err != nil {
		slog.Error("failed to compact data", "err", err)

		return err
	}

	buf.WriteByte('\n')

	_, err = conn.Write(buf.Bytes())
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return err
		}

		slog.Error("failed to send ping", "err", err)

		return err
	}

	slog.Debug("ping sent")

	return nil
}

func registerClient(conn net.Conn) error {
	addr := conn.RemoteAddr().String()

	slog.Info("client connected", "addr", addr)

	id := generateID()

	ActiveClients.Store(id, conn)

	data := &types.Data{
		Type: types.RegisterClient,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal data", "err", err)

		return err
	}

	jsonBytes = append(jsonBytes, []byte{'\r', '\n'}...)

	_, err = conn.Write(jsonBytes)
	if err != nil {
		slog.Error("failed to write data", "err", err)

		return err
	}

	return nil
}

var chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func generateID() string {
	return gonanoid.MustGenerate(chars, 6)
}
