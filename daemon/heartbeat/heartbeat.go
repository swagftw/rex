package heartbeat

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/swagftw/rex/utils"
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
	msgChan := make(chan []byte, 10)

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

			buf := &bytes.Buffer{}

			emptyLinesFound := 0

			for {
				// end of the message
				if emptyLinesFound == 2 {
					msgChan <- buf.Bytes()
					buf.Reset()
					emptyLinesFound = 0
				}

				bytesData, err := reader.ReadBytes('\n')
				if err != nil {
					if errors.Is(err, io.EOF) {
						return
					}

					slog.Error("failed to read the bytes", "err", err)

					return
				}

				buf.Write(bytesData)

				// check for empty line
				if bytes.Equal(bytesData, []byte{'\r', '\n'}) {
					emptyLinesFound++

					continue
				}
			}
		}(conn)
	}
}

func PingClient(conn net.Conn) error {
	buf := &bytes.Buffer{}

	buf.WriteString("REX PING\r\n")

	utils.GetRexReqBody(buf)

	_, err := conn.Write(buf.Bytes())
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

func handleConnMsg(conn net.Conn, msgChan chan []byte) {
	time.Sleep(time.Second)
	for msg := range msgChan {
		if bytes.HasPrefix(msg, []byte("REX REGISTER")) {
			err := registerClient(conn)
			if err != nil {
				return
			}

			continue
		}
	}
}

func registerClient(conn net.Conn) error {
	addr := conn.RemoteAddr().String()

	slog.Info("client connected", "addr", addr)

	id := utils.GenerateID()

	// ActiveClients.Store(id, conn)

	buf := &bytes.Buffer{}

	buf.WriteString("REX REGISTER")
	buf.WriteByte(' ')
	buf.WriteString(id)

	utils.GetRexReqBody(buf)

	_, err := conn.Write(buf.Bytes())
	if err != nil {
		slog.Error("failed to send client registered message", "err", err)

		return err
	}

	return nil
}
