package database

import (
	"MangaHub/pkg/models"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func SeedMangaFromJSON(path string) error {
	if path == "" {
		return fmt.Errorf("seed path is empty")
	}

	// check if the seed file exists before reading it
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Seed file not found: %s", path)
			return nil
		}
		return err
	}

	// check if manga table already has data to avoid duplicate seeding
	var count int
	if err := DB.QueryRow("SELECT COUNT(1) FROM manga").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		log.Printf("Seed skipped: manga table already has data")
		return nil
	}

	// read the seed file
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// unmarshal JSON into manga slice
	var items []models.Manga
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	if len(items) == 0 {
		log.Printf("Seed file is empty: %s", path)
		return nil
	}

	// insert manga entries into the database using a transaction
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback() // rollback on error, commit on success
	}()

	// use prepared statement
	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO manga (
			id,
			title,
			author,
			genres,
			status,
			total_chapters,
			description,
			cover_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// insert each manga item into the database
	for _, item := range items {
		// if required fields are missing, skip this item
		if item.ID == "" || item.Title == "" || item.Author == "" {
			log.Printf("Skipping manga item with missing required fields: %+v", item)
			continue
		}

		// convert genres slice to JSON string for storage
		genresJSON, err := json.Marshal(item.Genres)
		if err != nil {
			return err
		}

		//execute the prepared statement with manga item data
		if _, err := stmt.Exec(
			item.ID,
			item.Title,
			item.Author,
			string(genresJSON),
			item.Status,
			item.TotalChapters,
			item.Description,
			item.CoverURL,
		); err != nil {
			return err
		}
	}

	// commit the transaction to save changes to the database
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("Seeded %d manga entries", len(items))
	return nil
}
