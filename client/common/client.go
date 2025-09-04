package common

import (
	"bufio"
	"context"
	"io"
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
	MaxSizeAmount int
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

// StartClientLoop Send messages all the bets to the server
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

	// Create the connection to the server
	r := c.createClientSocket()
	if r != nil {
		log.Errorf("action: create_socket | result: fail | client_id: %v | error: %v",
			c.config.ID,
			r)
		return
	}
	defer c.closeSocket()

	// Send all bets from file
	c.sendBets()
}

func (c *Client) sendBets() error {
	file, err := getBetFile(c.config.ID)
	if err != nil {
		log.Errorf("action: open_file | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err)
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	max_payload_size := c.config.MaxSizeAmount - BATCH_HEADER
	buffer := make([]byte, 0, max_payload_size)
	for {
		// Get the next batch of bets to send
		batch, lastBet, err := getBetBatchToSend(reader, c.config, buffer)

		// Send the batch to the server
		sendErr := sendMessage(c.conn, batch)
		if sendErr != nil {
			break
		}

		// Wait for server response to ensure that the message was received and keep sending
		_, readErr := getResponse(c.conn)
		if readErr != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				readErr,
			)
			return readErr
		}

		// If all the bets were sent, exit the loop
		if err == io.EOF {
			break
		}

		// If there was a bet that didn't fit in the batch, keep it for the next iteration
		if lastBet != nil {
			buffer = make([]byte, 0, max_payload_size)
			buffer = append(buffer, lastBet...)
		} else {
			buffer = make([]byte, 0, max_payload_size)
		}
	}

	return nil
}
