package common

import (
	"bufio"
	"fmt"
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

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// Handle SIGINT and SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	c.handle_signals(sigChan)

	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Verify if the shutdown signal was received
		if !c.validateStillAlive() {
			break
		}

		// Create the connection the server in every loop iteration. Send an
		r := c.createClientSocket()
		if r != nil {
			log.Errorf("action: create_socket | result: fail | client_id: %v | error: %v",
				c.config.ID,
				r)
			return
		}

		// TODO: Modify the send to avoid short-write
		fmt.Fprintf(
			c.conn,
			"[CLIENT %v] Message N°%v\n",
			c.config.ID,
			msgID,
		)
		if !c.validateStillAlive() {
			break
		}

		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		c.conn.Close()
		if !c.validateStillAlive() {
			break
		}

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
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
