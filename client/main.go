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
	"github.com/swagftw/rex/utils"
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

	msgChan := make(chan []byte, 10)

	go processMsg(msgChan)

	for {
		err = readConnection(reader, msgChan)
		if errors.Is(err, io.EOF) {
			return
		}
	}
}

func registerClient(conn net.Conn) error {
	buf := new(bytes.Buffer)

	buf.WriteString("REX REGISTER\r\n")
	utils.GetRexReqBody(buf)

	_, err := conn.Write(buf.Bytes())
	if err != nil {
		slog.Error("failed to send client registered message", "err", err)

		return err
	}

	buf.Reset()

	reader := bufio.NewReader(conn)
	emptyLinesFound := 0

	for {
		if emptyLinesFound == 2 {
			if bytes.HasPrefix(buf.Bytes(), []byte("REX REGISTER")) {
				break
			}

			emptyLinesFound = 0
		}

		// wait till the client is registered
		bytesData, err := reader.ReadBytes('\n')
		if err != nil {
			slog.Error("failed to read the bytes", "err", err)

			return err
		}

		buf.Write(bytesData)

		if bytes.Equal(bytesData, []byte{'\r', '\n'}) {
			emptyLinesFound++
		}
	}

	slog.Info("registered client")

	return nil
}

func readConnection(reader *bufio.Reader, msgChan chan []byte) error {
	buf := &bytes.Buffer{}

	emptyLinesFound := 0

	for {
		// end of the message
		if emptyLinesFound == 2 {
			msgChan <- buf.Bytes()
			buf.Reset()
			emptyLinesFound = 0
		}

		// read line
		byteData, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Debug("connection closed", "err", err)

				return err
			}

			slog.Error("failed to read data", "err", err)

			return err
		}

		buf.Write(byteData)

		// end of the message
		if bytes.Equal(byteData, []byte{'\r', '\n'}) {
			emptyLinesFound++

			continue
		}
	}
}

func processMsg(msgChan chan []byte) {
	for msg := range msgChan {
		bytes.CutSuffix(msg, []byte{'\r', '\n', '\r', '\n'})
		slog.Info("received message", "msg", string(msg))
	}
}
