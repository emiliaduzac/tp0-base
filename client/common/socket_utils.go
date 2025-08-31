package common

import (
	"net"
)

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
