package common

import (
	"bufio"
	"io"
	"net"
	"os"
	"strings"
)

const IS_LAST_SIZE = 1
const BATCH_HEADER_SIZE = 2
const HEADER_SIZE = 6

type ProtocolError struct {
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}

func getBetBatchToSend(file *os.File, config ClientConfig) ([]byte, error) {
	max_payload_size := config.MaxSizeAmount - (IS_LAST_SIZE + BATCH_HEADER_SIZE) // 2 byte for total length and 1 byte for is_last
	if max_payload_size <= 0 {
		return nil, &ProtocolError{"Max payload size too small"}
	}
	batch := make([]byte, 0, max_payload_size)
	betCount := 0
	reader := bufio.NewReader(file)

	for {
		// If I reached the max number of bets, return the batch
		if betCount >= config.MaxBetsAmount {
			return serializeBatch(batch, false), nil
		}

		// guardo la ult posicion x si la nueva linea no entra en el batch
		lastPos, seekErr := file.Seek(0, io.SeekCurrent)
		if seekErr != nil {
			continue
		}

		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		log.Info("Leo linea: " + line)
		// readErr EOF y linea vacia -> devuelvo lo que tengo y EOF
		if len(line) == 0 && readErr == io.EOF {
			return serializeBatch(batch, true), readErr
		}

		if line == "" {
			continue
		}

		serializedBet, betErr := getSerializedBet(line, config.ID)
		if betErr != nil { // ProtocolError
			log.Error(
				"action: serialize_bet | result: fail | client_id: %v | error: %v",
				config.ID,
				betErr,
			)
			continue
		}

		if len(serializedBet) > max_payload_size {
			continue // bet demasiado grande, la ignoro
		}

		// complete un batch, devuelvo para mandar
		if len(batch)+len(serializedBet) > max_payload_size {
			_, seekErr = file.Seek(lastPos, io.SeekStart)
			if seekErr != nil {
				return serializeBatch(batch, false), nil
			}
			reader.Reset(file)
			return serializeBatch(batch, false), nil
		}

		betCount++
		batch = append(batch, serializedBet...)

		// fin archivo, devuelvo lo que tengo
		if readErr == io.EOF {
			betsBatch := serializeBatch(batch, true)
			return betsBatch, readErr
		}
	}
}

func serializeBatch(batch []byte, isLastBatch bool) []byte {
	lenBatch := len(batch)

	betsBatch := make([]byte, lenBatch+(IS_LAST_SIZE+BATCH_HEADER_SIZE))
	betsBatch[0] = byte(0)
	if isLastBatch {
		betsBatch[0] = 1
	}
	betsBatch[1] = byte(lenBatch >> 8)
	betsBatch[2] = byte(lenBatch & 0x00FF)
	copy(betsBatch[(IS_LAST_SIZE+BATCH_HEADER_SIZE):], batch)

	log.Info("Serializo un batch")
	return betsBatch
}

func getResponse(conn net.Conn) (byte, error) {
	return bufio.NewReader(conn).ReadByte()
}

func getBetFile(id string) (*os.File, error) {
	file, err := os.Open("/data/agency-" + id + ".csv")
	if err != nil {
		return nil, err
	}
	return file, nil
}
