package repository

import (
	"fmt"
	"github.com/dinerozz/web-behavior-backend/config"
	"github.com/jmoiron/sqlx"
	"log"
	"time"
)

func NewRepository(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Println("❌ Error connecting to database:", err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		log.Println("❌ Error pinging database:", err)
		return nil, err
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("✅ Connected to database")

	return db, nil
}
