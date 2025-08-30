package common

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Bet Struct that represents a bet
type Bet struct {
	Agency    string
	LastName  string
	Name      string
	Document  int
	Birthdate string
	Number    int
}

// getBet reads the bet information from environment variables and returns a Bet struct
func getBet(cliId string) Bet {
	fmt.Printf("ENV raw: NOMBRE=%q APELLIDO=%q DOCUMENTO=%q NACIMIENTO=%q NUMERO=%q\n",
		os.Getenv("NOMBRE"), os.Getenv("APELLIDO"), os.Getenv("DOCUMENTO"),
		os.Getenv("NACIMIENTO"), os.Getenv("NUMERO"))

	v := viper.New()
	v.AutomaticEnv()
	v.BindEnv("nombre", "NOMBRE")
	v.BindEnv("apellido", "APELLIDO")
	v.BindEnv("dni", "DOCUMENTO")
	v.BindEnv("nacimiento", "NACIMIENTO")
	v.BindEnv("numero", "NUMERO")

	fmt.Printf("--- Bet to send: %s, %s, %s, %d, %s, %d ---\n", cliId, v.GetString("apellido"), v.GetString("nombre"), v.GetInt("dni"), v.GetString("nacimiento"), v.GetInt("numero"))

	return Bet{
		Agency:    cliId,
		LastName:  v.GetString("apellido"),
		Name:      v.GetString("nombre"),
		Document:  v.GetInt("dni"),
		Birthdate: v.GetString("nacimiento"),
		Number:    v.GetInt("numero"),
	}
}

// serializeBet converts a Bet struct into a string for transmission
func serializeBet(bet Bet) string {
	return fmt.Sprintf("%s,%s,%s,%d,%s,%d",
		bet.Agency,
		bet.LastName,
		bet.Name,
		bet.Document,
		bet.Birthdate,
		bet.Number)
}
