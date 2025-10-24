package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Address                   string
	StakeHolderServiceAddress string
	AuthServiceAddress        string
	BlogServiceAddress        string
	FollowerServiceAddress    string
	TourServiceAddress        string
	PaymentServiceAddress     string
}

func GetConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Error while loading .env file")
	}

	return Config{
		StakeHolderServiceAddress: os.Getenv("STAKEHOLDER_SERVICE_ADDRESS"),
		AuthServiceAddress:        os.Getenv("AUTH_SERVICE_ADDRESS"),
		BlogServiceAddress:        os.Getenv("BLOG_SERVICE_ADDRESS"),
		FollowerServiceAddress:    os.Getenv("FOLLOWER_SERVICE_ADDRESS"),
		TourServiceAddress:        os.Getenv("TOUR_SERVICE_ADDRESS"),
		PaymentServiceAddress:     os.Getenv("PAYMENT_SERVICE_ADDRESS"),
		Address:                   os.Getenv("GATEWAY_ADDRESS"),
	}
}
