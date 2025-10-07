package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Address                   string
	StakeHolderServiceAddress string
}

func GetConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Error while loading .env file")
	}

	return Config{
		StakeHolderServiceAddress: os.Getenv("STAKEHOLDER_SERVICE_ADDRESS"),
		Address:                   os.Getenv("GATEWAY_ADDRESS"),
	}
}
