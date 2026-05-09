package models

type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"` //JSON array of as text
	Status        string   `json:"status"` // "ongoing", "completed", "hiatus"
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url"` // URL to manga cover image
}
