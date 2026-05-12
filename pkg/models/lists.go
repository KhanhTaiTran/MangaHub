package models

import "time"

type UserList struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type UserListItem struct {
	UserID         string    `json:"user_id"`
	ListID         string    `json:"list_id"`
	MangaID        string    `json:"manga_id"`
	CurrentChapter int       `json:"current_chapter"`
	Status         string    `json:"status"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type NotificationPrefs struct {
	UserID    string    `json:"user_id"`
	PrefsJSON string    `json:"prefs_json"`
	UpdatedAt time.Time `json:"updated_at"`
}
