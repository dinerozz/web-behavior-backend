package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port    string
	BaseURL string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type Config struct {
	Server ServerConfig
	DB     DatabaseConfig
	Env    string
}

func LoadConfig() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	return &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", "8080"),
			BaseURL: getEnv("BASE_URL", "http://localhost:8080"),
		},
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "test"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "dinero"),
			Password: getEnv("DB_PASS", "test"),
			DBName:   getEnv("DB_NAME", "expense_tracker_test"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Env: getEnv("ENV", "prod"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
