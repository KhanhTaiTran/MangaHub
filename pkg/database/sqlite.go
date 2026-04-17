package database

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

var DB *sql.DB // global variable to hold the database connection

func InitDB() {
	dbPath := os.Getenv("DB_PATH")

	if dbPath == "" {
		dbPath = "mangahub.db" // default path if not set in environment
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Set SQLite pragmas for better performance and reliability
	_, err = DB.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
	`)
	if err != nil {
		log.Fatal("Failed to set SQLite pragmas:", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to the Database at: ", dbPath)
}

func CreateTables(schema string) {
	_, err := DB.Exec(schema)
	if err != nil {
		log.Fatal("Error: ", err)
	}
}
