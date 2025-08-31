package common

import (
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
		c.closeSocket()
		log.Debugf("action: shutdown | result: success | signal: %v", ctx.Err())
		return
	default:
	}

	// Get the serialized bet_pck message to send, using the env variables
	bet_pck := getBetPacket(c.config.ID)
	betMsg := serializeBet(bet_pck)

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
	msg, readErr := getAck(c.conn)
	c.conn.Close()
	if readErr != nil || msg != 0 {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			readErr,
		)
		return
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
		bet_pck.Document,
		bet_pck.Number,
	)
}
