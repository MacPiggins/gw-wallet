package config

import (
	"os"

	"github.com/joho/godotenv"
)

const (
	defaultNatsConnstr = "nats://localhost:4222"
	defaultNatsStream = "wallet"
	defaultStorageConnstr = "localhost:5432"
	defaultServerAddr     = "localhost:8080"
	defaultExchangeAddr   = "localhost:8081"
	defaultAuthSecret     = "development-secret"
)

type Config struct {
	NatsConnstr    string
	NatsStream     string
	StorageConnstr string
	ServerAddr     string
	ExchangeAddr   string
	AuthSecret     string
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Load(files ...string) (*Config, error) {
	if err := godotenv.Load(files...); err != nil {
		return nil, err
	}
	return &Config{
			NatsConnstr:    envOrDefault("NATSCONNSTR", defaultNatsConnstr),
			NatsStream:     envOrDefault("NATSSTREAM", defaultNatsStream),
			StorageConnstr: envOrDefault("STORAGECONNSTR", defaultStorageConnstr),
			ServerAddr:     envOrDefault("SERVERADDR", defaultServerAddr),
			ExchangeAddr:   envOrDefault("EXCHANGEADDR", defaultExchangeAddr),
			AuthSecret:     envOrDefault("AUTHSECRET", defaultAuthSecret),
		},
		nil
}
