package common

import (
	"github.com/spf13/viper"
)

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

// getBetPacket reads the bet information from environment variables and returns a Bet struct
func getBetPacket(cliId string) BetPacket {
	v := viper.New()
	v.AutomaticEnv()
	v.BindEnv("nombre", "NOMBRE")
	v.BindEnv("apellido", "APELLIDO")
	v.BindEnv("dni", "DOCUMENTO")
	v.BindEnv("nacimiento", "NACIMIENTO")
	v.BindEnv("numero", "NUMERO")

	lastName := v.GetString("apellido")
	name := v.GetString("nombre")
	document := v.GetString("dni")
	birthdate := v.GetString("nacimiento")
	number := v.GetString("numero")

	return BetPacket{
		AgencyLen:    byte(len(cliId)),
		FirstNameLen: byte(len([]byte(name))),
		LastNameLen:  byte(len([]byte(lastName))),
		DocumentLen:  byte(len([]byte(document))),
		BirthdateLen: byte(len([]byte(birthdate))),
		NumberLen:    byte(len([]byte(number))),
		Agency:       cliId,
		FirstName:    name,
		LastName:     lastName,
		Document:     document,
		Birthdate:    birthdate,
		Number:       number,
	}
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
