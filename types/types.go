package types

type Data struct {
	Type int `json:"type"`
}

const (
	HeartbeatPing = iota
	HeartbeatPong
	RegisterClient
	Close
)
