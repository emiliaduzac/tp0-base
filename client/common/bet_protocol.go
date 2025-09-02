package common

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

const BATCH_HEADER = 3
const BET_HEADER = 6
const MAX_BATCH_SIZE = 8192

type OpCodeReq byte

const (
	OC_MORE_BATCHS OpCodeReq = 0
	OC_LAST_BATCH  OpCodeReq = 1
	OC_ASK_WINNERS OpCodeReq = 2
)

type OpCodeRes byte

const (
	OC_ACK     OpCodeRes = 0
	OC_NACK    OpCodeRes = 1
	OC_WINNERS OpCodeRes = 2
)

type ProtocolError struct {
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}

// Reads the client's csv file and returns a batch of bets to be sent.
// Receives: a bufio.Reader to read the file line by line and the client configuration
// Returns: a slice of bytes with the serialized batch, an error if something went wrong
// and a slice of bytes with the last bet read if this bet couldn't be added to the batch
func getBetBatchToSend(reader *bufio.Reader, config ClientConfig, batch []byte) ([]byte, []byte, error) {
	betsInBatch := 0

	for betsInBatch < config.MaxSizeAmount {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		// If the line is empty and we reached EOF, return what we have.
		if readErr == io.EOF && line == "" {
			return serializeBatch(batch, true), nil, readErr
		}
		// If its empty but not EOF, continue
		if line == "" {
			continue
		}

		// Serialize the current bet.
		serializedBet, betErr := getSerializedBet(line, config.ID)
		// If there was an error in the bet's format or its too large, skip it.
		if betErr != nil || len(serializedBet) > MAX_BATCH_SIZE {
			continue
		}

		// If this last bet didn't fit in the batch, return what we have and the bet to be sent later
		// in next batch
		if len(batch)+len(serializedBet) > MAX_BATCH_SIZE {
			return serializeBatch(batch, false), serializedBet, nil
		}

		batch = append(batch, serializedBet...)
		betsInBatch++

		// End of file, return the batch and indicate that its the last one
		if readErr == io.EOF {
			return serializeBatch(batch, true), nil, readErr
		}
	}

	return serializeBatch(batch, false), nil, nil
}

// Serialize a batch of bets adding the header
// Receives: a slice of bytes with the serialized bets and a boolean indicating if its the last batch
// Returns: a slice of bytes with the serialized batch
// Header: 1 byte to indicate if it's the last batch (0 no, 1 yes) + 2 bytes for the length of the batch
func serializeBatch(batch []byte, isLastBatch bool) []byte {
	lenBatch := len(batch)

	betsBatch := make([]byte, lenBatch+(BATCH_HEADER))
	betsBatch[0] = byte(OC_MORE_BATCHS)
	if isLastBatch {
		betsBatch[0] = byte(OC_LAST_BATCH)
	}
	betsBatch[1] = byte(lenBatch >> 8)
	betsBatch[2] = byte(lenBatch & 0x00FF)
	copy(betsBatch[(BATCH_HEADER):], batch)
	return betsBatch
}

// Reads the response from the server
func getResponseOpCode(conn net.Conn) (byte, error) {
	return bufio.NewReader(conn).ReadByte()
}

// func getResponse(conn net.Conn) ([]byte, error) {
//
// }

// Opens the client's csv file of bets
func getBetFile(id string) (*os.File, error) {
	file, err := os.Open("/data/agency-" + id + ".csv")
	if err != nil {
		return nil, err
	}
	return file, nil
}

func getWinnersRequest(id string) ([]byte, error) {
	n, err := strconv.ParseUint(id, 10, 8)
	if err != nil {
		return nil, &ProtocolError{Message: fmt.Sprintf("Client ID %v is not valid", id)}
	}
	return []byte{byte(OC_ASK_WINNERS), byte(n)}, nil
}
