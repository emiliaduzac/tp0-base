package common

import (
	"bufio"
	"io"
	"net"
	"os"
	"strings"
)

type ProtocolError struct {
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}

// Packet structure for a bet.
// First 6 bytes are lengths of the respective fields
// followed by the actual data. Total length can be calculated by adding the firts 6 bytes
// together and adding 6 to it.
type BetPacket struct {
	// Lengths
	AgencyLen    byte
	FirstNameLen byte
	LastNameLen  byte
	DocumentLen  byte
	BirthdateLen byte
	NumberLen    byte
	// Fields
	Agency    string
	FirstName string
	LastName  string
	Document  string
	Birthdate string
	Number    string
}

func getBetBatch(file *os.File, config ClientConfig) ([]byte, error) {
	batch := make([]byte, 0, config.MaxSizeAmount)
	betCount := 0
	reader := bufio.NewReader(file)

	for {
		// guardo la ult posicion x si la nueva linea no entra en el batch
		lastPos, seekErr := file.Seek(0, io.SeekCurrent)
		if seekErr != nil {
			return batch, seekErr
		}

		// leo linea (== bet)
		line, readErr := reader.ReadString('\n')
		// readErr que no es EOF -> devuelvo hasta donde llegue y el error
		// readErr EOF y linea vacia -> devuelvo lo que tengo y EOF
		if readErr != nil && readErr != io.EOF || (len(line) == 0 && readErr == io.EOF) {
			return batch, readErr
		}
		// linea vacia, sigo
		if line == "" {
			continue
		}

		// obtengo la bet y la serializo
		bet, betErr := getBetPacket(strings.TrimSuffix(line, "\n"), config.ID)
		if betErr != nil {
			return batch, betErr
		}
		serializedBet := serializeBet(*bet)

		if len(serializedBet) > config.MaxSizeAmount {
			return batch, &ProtocolError{Message: "Bet size exceeds maximum batch size"}
		}

		// complete un batch, devuelvo para mandar
		if betCount >= config.MaxBetsAmount || len(batch)+len(serializedBet) > config.MaxSizeAmount {
			_, seekErr = file.Seek(lastPos, io.SeekStart)
			if seekErr != nil {
				return batch, seekErr
			}
			reader.Reset(file)
			return batch, nil
		}

		betCount++
		batch = append(batch, serializedBet...)

		// fin archivo, devuelvo lo que tengo
		if readErr == io.EOF {
			return batch, nil
		}
	}
}

func getBetPacket(line string, cliId string) (*BetPacket, error) {
	fields := strings.Split(line, ",")
	if len(fields) != 5 {
		return nil, &ProtocolError{Message: "Invalid bet format"}
	}

	return &BetPacket{
		AgencyLen:    byte(len([]byte(cliId))),
		FirstNameLen: byte(len([]byte(fields[0]))),
		LastNameLen:  byte(len([]byte(fields[1]))),
		DocumentLen:  byte(len([]byte(fields[2]))),
		BirthdateLen: byte(len([]byte(fields[3]))),
		NumberLen:    byte(len([]byte(fields[4]))),
		Agency:       cliId,
		FirstName:    fields[0],
		LastName:     fields[1],
		Document:     fields[2],
		Birthdate:    fields[3],
		Number:       fields[4],
	}, nil
}

// serializeBet converts a BetPacket into a slice of bytes for transmission
func serializeBet(betPacket BetPacket) []byte {
	totalLen := 6 + betPacket.AgencyLen + betPacket.FirstNameLen + betPacket.LastNameLen +
		betPacket.DocumentLen + betPacket.BirthdateLen + betPacket.NumberLen

	betMsg := make([]byte, totalLen)
	betMsg[0] = betPacket.AgencyLen
	betMsg[1] = betPacket.FirstNameLen
	betMsg[2] = betPacket.LastNameLen
	betMsg[3] = betPacket.DocumentLen
	betMsg[4] = betPacket.BirthdateLen
	betMsg[5] = betPacket.NumberLen

	off := 6
	off += copy(betMsg[off:], betPacket.Agency)
	off += copy(betMsg[off:], betPacket.FirstName)
	off += copy(betMsg[off:], betPacket.LastName)
	off += copy(betMsg[off:], betPacket.Document)
	off += copy(betMsg[off:], betPacket.Birthdate)
	copy(betMsg[off:], betPacket.Number)

	return betMsg
}

func getAck(conn net.Conn) (byte, error) {
	return bufio.NewReader(conn).ReadByte()
}
