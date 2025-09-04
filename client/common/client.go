package common

import (
	"bufio"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
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
	config         ClientConfig
	conn           net.Conn
	keepingAlive   bool
	keepAliveMutex sync.RWMutex
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config:       config,
		keepingAlive: true,
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	c.handle_signals(sigChan)

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
	signal.Stop(sigChan)
	close(sigChan)
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
		if sendErr != nil || !c.validateStillAlive() {
			break
		}

		// Wait for server response to ensure that the message was received and keep sending
		_, readErr := getResponse(c.conn)
		if !c.validateStillAlive() {
			break
		}
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

func (c *Client) close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) shutdown(err string) {
	c.keepAliveMutex.Lock()
	c.keepingAlive = false
	c.keepAliveMutex.Unlock()
	log.Debugf("action: shutdown | result: in_progress | signal: %v", err)
	c.close()
	log.Debugf("action: shutdown | result: success | signal: %v", err)
}

func (c *Client) handle_signals(sigChan chan os.Signal) {
	go func() {
		sig := <-sigChan
		switch sig {
		case syscall.SIGINT:
			c.shutdown("SIGINT")
		case syscall.SIGTERM:
			c.shutdown("SIGTERM")
		default:
		}
	}()
}

func (c *Client) validateStillAlive() bool {
	c.keepAliveMutex.RLock()
	if !c.keepingAlive {
		c.keepAliveMutex.RUnlock()
		c.close()
		return false
	}
	c.keepAliveMutex.RUnlock()
	return true
}
