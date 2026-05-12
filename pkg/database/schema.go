package database

import (
	"database/sql"
	"fmt"
	"log"
)

const baseSchema = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS manga (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	author TEXT NOT NULL,
	genres TEXT NOT NULL,
	status TEXT NOT NULL,
	total_chapters INTEGER NOT NULL,
	description TEXT,
	cover_url TEXT
);

CREATE TABLE IF NOT EXISTS user_progress (
	user_id TEXT NOT NULL,
	manga_id TEXT NOT NULL,
	current_chapter INTEGER DEFAULT 0,
	status TEXT DEFAULT 'reading',
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (user_id, manga_id),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_lists (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(user_id, name),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_list_items (
	user_id TEXT NOT NULL,
	list_id TEXT NOT NULL,
	manga_id TEXT NOT NULL,
	current_chapter INTEGER DEFAULT 0,
	status TEXT DEFAULT 'reading',
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (user_id, list_id, manga_id),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (list_id) REFERENCES user_lists(id) ON DELETE CASCADE,
	FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_notification_prefs (
	user_id TEXT PRIMARY KEY,
	prefs_json TEXT NOT NULL,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

func InitSchema() {
	ExecuteSchema(baseSchema)
	if err := addColumnIfMissing("manga", "cover_url", "TEXT"); err != nil {
		log.Printf("Schema migration skipped: %v", err)
	}
}

func addColumnIfMissing(table, column, columnType string) error {
	exists, err := hasColumn(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = DB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType))
	return err
}

// helper function to check if a column exists in a table using PRAGMA table_info
func hasColumn(table, column string) (bool, error) {
	rows, err := DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}
