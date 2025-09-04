package common

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

// Reads the client's csv file and returns a batch of bets to be sent.
// Receives: a bufio.Reader to read the file line by line and the client configuration
// Returns: a slice of bytes with the serialized batch, an error if something went wrong
// and a slice of bytes with the last bet read if this bet couldn't be added to the batch
func getBetBatchToSend(reader *bufio.Reader, config ClientConfig, batch []byte) ([]byte, []byte, error) {
	betsInBatch := 0

	for betsInBatch < config.MaxBetsAmount {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		// If the line is empty and we reached EOF, return what we have.
		if readErr == io.EOF && line == "" {
			if betsInBatch == 0 {
				return nil, nil, readErr
			}
			return serializeBatch(batch), nil, readErr
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
			return serializeBatch(batch), serializedBet, nil
		}

		batch = append(batch, serializedBet...)
		betsInBatch++

		// End of file, return the batch and indicate that its the last one
		if readErr == io.EOF {
			return serializeBatch(batch), nil, readErr
		}
	}

	return serializeBatch(batch), nil, nil
}

// Serialize a batch of bets adding the header
// Receives: a slice of bytes with the serialized bets and a boolean indicating if its the last batch
// Returns: a slice of bytes with the serialized batch
// Header: 1 byte to indicate if it a batch message (0) or an end message (1) + 2 bytes for the length of the batch
func serializeBatch(batch []byte) []byte {
	lenBatch := len(batch)

	betsBatch := make([]byte, lenBatch+(BATCH_HEADER))
	betsBatch[0] = byte(OpCodeRequest(OC_BATCHS))
	betsBatch[1] = byte(lenBatch >> 8)
	betsBatch[2] = byte(lenBatch & 0x00FF)
	copy(betsBatch[(BATCH_HEADER):], batch)
	return betsBatch
}

// Returns an end message to nofity that all batchs were sent
func getEndMessage(cliID string) []byte {
	n, _ := strconv.ParseUint(cliID, 10, 8)
	return []byte{byte(OC_END), byte(n)}
}

// Reads the response from the server
func getResponseOpCode(r *bufio.Reader) (byte, error) {
	return bufio.NewReader(r).ReadByte()
}

// Opens the client's csv file of bets
func getBetFile(id string) (*os.File, error) {
	file, err := os.Open("/data/agency-" + id + ".csv")
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Reads the server's response to find the winners of the lottery.
func parseWinnersResponse(r *bufio.Reader) (int, error) {
	totalLenBuf := make([]byte, WINNERS_HEADER)
	if lenRead, err := io.ReadFull(r, totalLenBuf); err != nil || lenRead != WINNERS_HEADER {
		return 0, err
	}

	totalLen := int(totalLenBuf[0])<<8 | int(totalLenBuf[1])

	totalWinners := 0
	totalRead := 0
	for totalRead < totalLen {
		dniLenBuf := make([]byte, DNI_HEADER)
		if lenLen, err := io.ReadFull(r, dniLenBuf); err != nil || lenLen != DNI_HEADER {
			return totalWinners, err
		}

		dniBuf := make([]byte, int(dniLenBuf[0]))
		if lenDni, err := io.ReadFull(r, dniBuf); err != nil || lenDni != int(dniLenBuf[0]) {
			return totalWinners, err
		}
		totalWinners++
		totalRead += 1 + int(dniLenBuf[0])
	}

	return totalWinners, nil
}
