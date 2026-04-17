package models

import "time"

type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	Password_hash string    `json:"-"` // never return password hash in API responses
	Created_at    time.Time `json:"created_at"`
}

type Manga struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Author         string   `json:"author"`
	Genres         []string `json:"genres"` //JSON array of as text
	Status         string   `json:"status"` // "ongoing", "completed", "hiatus"
	Total_chapters int      `json:"total_chapters"`
	Description    string   `json:"description"`
}

type User_progress struct {
	User_id         string    `json:"user_id"`
	Manga_id        string    `json:"manga_id"`
	Current_chapter int       `json:"current_chapter"`
	Status          string    `json:"status"` // "reading", "completed", "dropped"
	Updated_at      time.Time `json:"updated_at"`
}
