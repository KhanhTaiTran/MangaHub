package main

import (
	"MangaHub/pkg/database"
	"log"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS manga (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	author TEXT NOT NULL,
	genres TEXT NOT NULL, 
	status TEXT NOT NULL,
	total_chapters INTEGER NOT NULL,
	description TEXT
);

CREATE TABLE IF NOT EXISTS user_progress (
	user_id TEXT NOT NULL,
	manga_id TEXT NOT NULL,
	current_chapter INTEGER NOT NULL,
	status TEXT NOT NULL,
	updated_at DATETIME NOT NULL,
	PRIMARY KEY (user_id, manga_id),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
);
`

func main() {
	// Initialize database connection
	database.InitDB()

	// Create tables if they don't exist
	database.ExecuteSchema(schema)

	log.Println("Database initialized and tables created (if not exist). Starting API server...")
}
