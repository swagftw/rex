package types

type Data struct {
	Type int `json:"type"`
	Msg  any `json:"msg"`
}

type RegisterClientData struct {
	ClientAddr string `json:"clientAddr"`
}

const (
	HeartbeatPing = iota
	HeartbeatPong
	RegisterClient
	Close
	ClientRegistered
)
