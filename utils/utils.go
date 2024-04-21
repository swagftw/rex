package utils

import (
	"bytes"

	gonanoid "github.com/matoous/go-nanoid"
)

func GetRexReqBody(buf *bytes.Buffer) {
	buf.WriteByte('\r')
	buf.WriteByte('\n')
	buf.WriteByte('\r')
	buf.WriteByte('\n')
}

var chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func GenerateID() string {
	return gonanoid.MustGenerate(chars, 6)
}
