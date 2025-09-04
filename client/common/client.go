package common

import (
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

// StartClient Send messages to the client until some time threshold is met
func (c *Client) StartClient() {
	// Handle SIGINT and SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	c.handle_signals(sigChan)

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
	if !c.validateStillAlive() {
		return
	}
	if sendErr != nil {
		log.Errorf("action: apuesta_enviada | result: fail | dni: %v | numero: %v",
			c.config.ID,
			sendErr,
		)
		return
	}

	// Wait for server response to ensure that the message was received
	msg, readErr := getAck(c.conn)
	if !c.validateStillAlive() {
		return
	}
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
	signal.Stop(sigChan)
	close(sigChan)
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
