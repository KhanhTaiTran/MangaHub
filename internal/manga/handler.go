package manga

import (
	"MangaHub/pkg/database"
	"MangaHub/pkg/models"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type searchResponse struct {
	Items []models.Manga `json:"items"`
	Count int            `json:"count"`
}

// Search handler to search manga by title, author, genre, or status with pagination
func Search(c *gin.Context) {
	queryText := strings.TrimSpace(c.Query("q"))   // search by title
	author := strings.TrimSpace(c.Query("author")) // search by author
	genre := strings.TrimSpace(c.Query("genre"))   // search by genre
	status := strings.TrimSpace(c.Query("status")) // search by status: "ongoing", "completed", "hiatus"

	limit := parsePositiveInt(c.Query("limit"), 20, 100)     // default limit 20, max limit 100
	offset := parsePositiveInt(c.Query("offset"), 0, 100000) // default offset 0, max offset 100000

	// build dynamic WHERE clause based on provided query parameters
	where := []string{"1=1"}
	args := make([]interface{}, 0) // arguments for prepared statement

	// only add conditions for non-empty query parameters
	if queryText != "" {
		where = append(where, "lower(title) LIKE ?")
		args = append(args, "%"+strings.ToLower(queryText)+"%")
	}
	if author != "" {
		where = append(where, "lower(author) LIKE ?")
		args = append(args, "%"+strings.ToLower(author)+"%")
	}
	if genre != "" {
		where = append(where, "lower(genres) LIKE ?")
		args = append(args, "%"+strings.ToLower(genre)+"%")
	}
	if status != "" {
		where = append(where, "lower(status) = ?")
		args = append(args, strings.ToLower(status))
	}

	// add pagination parameters at the end of arguments
	args = append(args, limit, offset)

	query := "SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga WHERE " + strings.Join(where, " AND ") + " ORDER BY title LIMIT ? OFFSET ?"
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	// parse query results into manga slice
	items := make([]models.Manga, 0)
	for rows.Next() {
		var item models.Manga
		var genresText string
		if err := rows.Scan(&item.ID, &item.Title, &item.Author, &genresText, &item.Status, &item.TotalChapters, &item.Description, &item.CoverURL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		item.Genres = parseGenres(genresText)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	// return search results as JSON response
	c.JSON(http.StatusOK, searchResponse{
		Items: items,
		Count: len(items),
	})
}

// GetByID handler to fetch manga details by ID
func GetByID(c *gin.Context) {
	mangaID := strings.TrimSpace(c.Param("id")) // validate manga ID parameter
	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga id is required"})
		return
	}

	var item models.Manga
	var genresText string
	err := database.DB.QueryRow(
		"SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga WHERE id = ?",
		mangaID,
	).Scan(&item.ID, &item.Title, &item.Author, &genresText, &item.Status, &item.TotalChapters, &item.Description, &item.CoverURL)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "manga not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	item.Genres = parseGenres(genresText)
	c.JSON(http.StatusOK, item)
}

// helper function to parse genres from JSON text stored in database
func parseGenres(genresText string) []string {
	if genresText == "" {
		return []string{}
	}
	var genres []string
	if err := json.Unmarshal([]byte(genresText), &genres); err != nil {
		return []string{}
	}
	return genres
}

// helper function to parse a positive integer from a string with fallback and max limits
func parsePositiveInt(value string, fallback, max int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}
