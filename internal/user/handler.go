package user

import (
	"MangaHub/pkg/database"
	"MangaHub/pkg/models"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type addLibraryRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	ListName       string `json:"list_name"` // optional, default to "reading"
	Status         string `json:"status"`
	CurrentChapter int    `json:"current_chapter"`
}

type updateProgressRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	ListName       string `json:"list_name"`
	Status         string `json:"status"`
	CurrentChapter int    `json:"current_chapter"`
}

type libraryItem struct {
	ListName       string       `json:"list_name"`
	Manga          models.Manga `json:"manga"`
	CurrentChapter int          `json:"current_chapter"`
	Status         string       `json:"status"`
	UpdatedAt      string       `json:"updated_at"`
}

// add manga to user's library with progress and status
func AddToLibrary(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req addLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	listName := normalizeListName(req.ListName)
	status := normalizeStatus(req.Status)

	// validate manga exists before adding to library
	if !mangaExists(req.MangaID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "manga not found"})
		return
	}

	listID, err := ensureUserList(userID, listName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := upsertListItem(userID, listID, req.MangaID, req.CurrentChapter, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Added to library",
		"list_name":       listName,
		"manga_id":        req.MangaID,
		"current_chapter": req.CurrentChapter,
		"status":          status,
	})
}

// get user's library with optional filtering by list name and pagination
func GetLibrary(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	listName := strings.TrimSpace(c.Query("list_name"))
	query := `
		SELECT l.name, li.manga_id, li.current_chapter, li.status, li.updated_at,
			m.title, m.author, m.genres, m.status, m.total_chapters, m.description, m.cover_url
		FROM user_list_items li
		JOIN user_lists l ON l.id = li.list_id
		JOIN manga m ON m.id = li.manga_id
		WHERE li.user_id = ?
	`
	// build query args with optional list name filter
	args := []interface{}{userID}
	if listName != "" {
		query += " AND l.name = ?"
		args = append(args, listName)
	}
	query += " ORDER BY li.updated_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	// parse query results into library item slice
	items := make([]libraryItem, 0)
	for rows.Next() {
		var item libraryItem
		var genresText string
		if err := rows.Scan(
			&item.ListName,
			&item.Manga.ID,
			&item.CurrentChapter,
			&item.Status,
			&item.UpdatedAt,
			&item.Manga.Title,
			&item.Manga.Author,
			&genresText,
			&item.Manga.Status,
			&item.Manga.TotalChapters,
			&item.Manga.Description,
			&item.Manga.CoverURL,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		item.Manga.Genres = parseGenres(genresText)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"count": len(items),
	})
}

// update manga reading progress and status in user's library
func UpdateProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// validate request body and binding rules
	var req updateProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	listName := normalizeListName(req.ListName)
	status := normalizeStatus(req.Status)

	if !mangaExists(req.MangaID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "manga not found"})
		return
	}

	listID, err := ensureUserList(userID, listName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := upsertListItem(userID, listID, req.MangaID, req.CurrentChapter, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Progress updated",
		"list_name":       listName,
		"manga_id":        req.MangaID,
		"current_chapter": req.CurrentChapter,
		"status":          status,
	})
}

// helper function to ensure user list exists
func ensureUserList(userID, listName string) (string, error) {
	var listID string
	err := database.DB.QueryRow(
		"SELECT id FROM user_lists WHERE user_id = ? AND name = ?",
		userID,
		listName,
	).Scan(&listID)
	if err == nil {
		return listID, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	// create new list if not found
	listID = uuid.New().String()
	_, err = database.DB.Exec(
		"INSERT INTO user_lists (id, user_id, name, created_at) VALUES (?, ?, ?, ?)",
		listID,
		userID,
		listName,
		time.Now(),
	)
	if err != nil {
		return "", err
	}
	return listID, nil
}

// helper function to upsert list item with progress and status
func upsertListItem(userID, listID, mangaID string, currentChapter int, status string) error {
	if currentChapter < 0 {
		currentChapter = 0
	}
	// use upsert to insert or update list item in one query
	_, err := database.DB.Exec(`
		INSERT INTO user_list_items (
			user_id,
			list_id,
			manga_id,
			current_chapter,
			status,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, list_id, manga_id) DO UPDATE SET
			current_chapter = excluded.current_chapter,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, userID, listID, mangaID, currentChapter, status, time.Now())
	return err
}

// check if manga exists in database by ID
func mangaExists(mangaID string) bool {
	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM manga WHERE id = ?)", mangaID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// helper function to normalize list name with default and trimming
func normalizeListName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "reading"
	}
	return name
}

// helper function to normalize status with default and validation
func normalizeStatus(status string) string {
	status = strings.TrimSpace(status)
	switch status {
	case "reading", "completed", "dropped", "plan_to_read":
		return status
	case "":
		return "reading"
	default:
		return "reading"
	}
}

func parseGenres(genresText string) []string {
	// if genresText is empty, return empty slice
	if genresText == "" {
		return []string{}
	}

	// unmarshal JSON array of genres from text
	var genres []string
	if err := json.Unmarshal([]byte(genresText), &genres); err != nil {
		return []string{}
	}
	return genres
}
