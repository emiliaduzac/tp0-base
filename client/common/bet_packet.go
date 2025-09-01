package common

import "strings"

const TOTAL_FIELDS_BET = 5

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

func getBetPacket(line string, cliId string) (*BetPacket, error) {
	fields := strings.Split(line, ",")
	if len(fields) != TOTAL_FIELDS_BET {
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
	totalLen := HEADER_SIZE + betPacket.AgencyLen + betPacket.FirstNameLen + betPacket.LastNameLen +
		betPacket.DocumentLen + betPacket.BirthdateLen + betPacket.NumberLen

	betMsg := make([]byte, totalLen)
	betMsg[0] = betPacket.AgencyLen
	betMsg[1] = betPacket.FirstNameLen
	betMsg[2] = betPacket.LastNameLen
	betMsg[3] = betPacket.DocumentLen
	betMsg[4] = betPacket.BirthdateLen
	betMsg[5] = betPacket.NumberLen

	off := HEADER_SIZE
	off += copy(betMsg[off:], betPacket.Agency)
	off += copy(betMsg[off:], betPacket.FirstName)
	off += copy(betMsg[off:], betPacket.LastName)
	off += copy(betMsg[off:], betPacket.Document)
	off += copy(betMsg[off:], betPacket.Birthdate)
	copy(betMsg[off:], betPacket.Number)

	return betMsg
}

func getSerializedBet(line string, cliId string) ([]byte, error) {
	bet, betErr := getBetPacket(line, cliId)
	if betErr != nil {
		return nil, betErr
	}
	return serializeBet(*bet), nil
}
