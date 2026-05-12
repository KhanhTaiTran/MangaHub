package models

import "time"

type Users struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // never return password hash in API responses
	CreatedAt    time.Time `json:"created_at"`
}

type User_progress struct {
	UserID         string    `json:"user_id"`
	MangaID        string    `json:"manga_id"`
	CurrentChapter int       `json:"current_chapter"`
	Status         string    `json:"status"` // "reading", "completed", "dropped"
	UpdatedAt      time.Time `json:"updated_at"`
}
