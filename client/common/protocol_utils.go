package common

import (
	"net"
)

const BATCH_HEADER = 3
const BET_HEADER = 6
const MAX_BATCH_SIZE = 8192

type OpCodeRequest byte

const (
	OC_BATCHS OpCodeRequest = 0
	OC_END    OpCodeRequest = 1
)

type OpCodeResponse byte

const (
	OC_ACK     OpCodeResponse = 0
	OC_NACK    OpCodeResponse = 1
	OC_WINNERS OpCodeResponse = 2
)

type ProtocolError struct {
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}

// closes the client's connection if it's open
func (c *Client) closeSocket() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// sends a message to the server, ensuring that all bytes are sent
func sendMessage(conn net.Conn, betMsg []byte) error {
	totalSent := 0
	for totalSent < len(betMsg) {
		sent_bytes, err := conn.Write(betMsg[totalSent:])
		if err != nil {
			return err
		}
		totalSent += sent_bytes
	}

	return nil
}
