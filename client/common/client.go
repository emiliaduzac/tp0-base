package common

import (
	"bufio"
	"context"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}

	c.conn = conn
	log.Debugf("action: connect | result: success | client_id: %v", c.config.ID)
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// Handle SIGINT and SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	select {
	case <-ctx.Done():
		log.Debugf("action: shutdown | result: in_progress | signal: %v", ctx.Err())
		c.close()
		log.Debugf("action: shutdown | result: success | signal: %v", ctx.Err())
		return
	default:
	}

	// Get the serialized bet message to send, using the env variables
	bet := getBetPacket(c.config.ID)
	betMsg := serializeBet(bet)

	// Create the connection the server
	r := c.createClientSocket()
	if r != nil {
		log.Errorf("action: create_socket | result: fail | client_id: %v | error: %v",
			c.config.ID,
			r)
		return
	}

	// Send bet to the server
	sendErr := sendMessage(c.conn, betMsg)
	if sendErr != nil {
		log.Errorf("action: apuesta_enviada | result: fail | dni: %v | numero: %v",
			c.config.ID,
			sendErr,
		)
		return
	}

	// Wait for server response to ensure that the message was received
	msg, readErr := bufio.NewReader(c.conn).ReadByte()
	c.conn.Close()
	if readErr != nil || msg != byte(0) {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			readErr,
		)
		return
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
		bet.Document,
		bet.Number,
	)
}

// closes the client's connection if it's open
func (c *Client) close() {
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
