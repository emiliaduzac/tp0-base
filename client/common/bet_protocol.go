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
	max_size := config.MaxSizeAmount - (IS_LAST_SIZE + BATCH_HEADER_SIZE) // 2 byte for total length and 1 byte for is_last
	batch := make([]byte, 0, max_size)
	betCount := 0
	reader := bufio.NewReader(file)

	for len(batch) < max_size && betCount < config.MaxBetsAmount {
		// guardo la ult posicion x si la nueva linea no entra en el batch
		// lastPos, seekErr := file.Seek(0, io.SeekCurrent)
		// if seekErr != nil {
		// 	return batch, seekErr
		// }

		// leo linea (== bet)
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		// readErr que no es EOF -> devuelvo hasta donde llegue y el error
		if readErr != nil && readErr != io.EOF {
			betsBatch := serializeBatch(batch, false)
			return betsBatch, nil
		}
		// readErr EOF y linea vacia -> devuelvo lo que tengo y EOF
		if len(line) == 0 && readErr == io.EOF {
			betsBatch := serializeBatch(batch, true)
			return betsBatch, nil
		}
		// linea vacia, sigo
		if line == "" {
			continue
		}

		// obtengo la bet y la serializo
		bet, betErr := getBetPacket(line, config.ID)
		if betErr != nil {
			betsBatch := serializeBatch(batch, false)
			return betsBatch, nil
		}
		serializedBet := serializeBet(*bet)

		if len(serializedBet) > max_size {
			betsBatch := serializeBatch(batch, false)
			return betsBatch, nil
		}

		// complete un batch, devuelvo para mandar
		// if betCount >= config.MaxBetsAmount || len(batch)+len(serializedBet) > config.MaxSizeAmount {
		// 	_, seekErr = file.Seek(lastPos, io.SeekStart)
		// 	if seekErr != nil {
		// 		return batch, seekErr
		// 	}
		// 	reader.Reset(file)
		// 	return batch, nil
		// }

		betCount++
		batch = append(batch, serializedBet...)

		// fin archivo, devuelvo lo que tengo
		if readErr == io.EOF {
			betsBatch := serializeBatch(batch, true)
			return betsBatch, readErr
		}
	}

	betsBatch := serializeBatch(batch, false)
	return betsBatch, nil
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
