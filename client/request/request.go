package request

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func GetClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: 20,
		},
	}
}

func Request(wg *sync.WaitGroup, client *http.Client) {
	defer wg.Done()

	req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		slog.Error("failed to create request", "err", err)

		return
	}

	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to send request", "err", err)

		return
	}

	defer resp.Body.Close()

	if !IsSuccess(resp) {
		slog.Error("request failed", "status", resp.Status)

		return
	}

	body, err := readBody(resp.Body)
	if err != nil {
		return
	}

	slog.Info("request succeeded", "status", resp.Status, "body", string(body))
}

func readBody(reader io.Reader) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	defer bufPool.Put(buf)

	_, err := io.Copy(buf, reader)
	if err != nil {
		slog.Error("failed to read response body", "err", err)

		return nil, err
	}

	return buf.Bytes(), nil
}

func IsSuccess(resp *http.Response) bool {
	return resp.StatusCode < 300 && resp.StatusCode >= 200
}
