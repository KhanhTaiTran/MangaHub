package grpc

import (
	"MangaHub/pkg/database"
	"MangaHub/proto/mangahubpb"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Server implements the gRPC services for MangaHub, including manga retrieval, search, user profile, and library management
type Server struct {
	mangahubpb.UnimplementedMangaServiceServer
	mangahubpb.UnimplementedUserServiceServer
}

// create new Server instance with empty struct
func NewServer() *Server {
	return &Server{}
}

// GetManga retrieves manga details by ID, returning a Manga object or an error if not found or if there's a database issue
func (s *Server) GetManga(ctx context.Context, req *mangahubpb.GetMangaRequest) (*mangahubpb.Manga, error) {
	mangaID := strings.TrimSpace(req.GetId())
	if mangaID == "" {
		return nil, status.Error(codes.InvalidArgument, "manga id is required")
	}

	var item mangahubpb.Manga
	var genresText string
	row := database.DB.QueryRow(
		"SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga WHERE id = ?",
		mangaID,
	)
	// scan the result into the item struct
	if err := row.Scan(
		&item.Id,
		&item.Title,
		&item.Author,
		&genresText,
		&item.Status,
		&item.TotalChapters,
		&item.Description,
		&item.CoverUrl,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "manga not found")
		}
		return nil, status.Error(codes.Internal, "database error")
	}
	item.Genres = parseGenres(genresText)
	return &item, nil
}

// SearchManga searches for manga based on the provided criteria, returning a list of matching manga or an error if there's a database issue.
func (s *Server) SearchManga(ctx context.Context, req *mangahubpb.SearchRequest) (*mangahubpb.SearchResponse, error) {
	queryText := strings.TrimSpace(req.GetQuery())
	author := strings.TrimSpace(req.GetAuthor())
	genre := strings.TrimSpace(req.GetGenre())
	statusFilter := strings.TrimSpace(req.GetStatus())

	limit := clampInt(int(req.GetLimit()), 20, 100)
	offset := clampInt(int(req.GetOffset()), 0, 100000)

	where := []string{"1=1"}
	args := make([]interface{}, 0)
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
	if statusFilter != "" {
		where = append(where, "lower(status) = ?")
		args = append(args, strings.ToLower(statusFilter))
	}
	args = append(args, limit, offset)

	query := "SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga WHERE " +
		strings.Join(where, " AND ") + " ORDER BY title LIMIT ? OFFSET ?"
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	defer rows.Close()

	items := make([]*mangahubpb.Manga, 0)
	for rows.Next() {
		var item mangahubpb.Manga
		var genresText string
		if err := rows.Scan(
			&item.Id,
			&item.Title,
			&item.Author,
			&genresText,
			&item.Status,
			&item.TotalChapters,
			&item.Description,
			&item.CoverUrl,
		); err != nil {
			return nil, status.Error(codes.Internal, "database error")
		}
		item.Genres = parseGenres(genresText)
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	return &mangahubpb.SearchResponse{Items: items, Count: int32(len(items))}, nil
}

// UpdateProgress updates the reading progress of a manga for the authenticated user, returning a confirmation or an error if there's a database issue.
func (s *Server) UpdateProgress(ctx context.Context, req *mangahubpb.ProgressRequest) (*mangahubpb.ProgressResponse, error) {
	userID, _, err := extractUser(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	mangaID := strings.TrimSpace(req.GetMangaId())
	if mangaID == "" {
		return nil, status.Error(codes.InvalidArgument, "manga id is required")
	}
	if !mangaExists(mangaID) {
		return nil, status.Error(codes.NotFound, "manga not found")
	}

	listName := normalizeListName(req.GetListName())
	statusValue := normalizeStatus(req.GetStatus())
	chapter := int(req.GetCurrentChapter())
	if chapter < 0 {
		chapter = 0
	}

	listID, err := ensureUserList(userID, listName)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	if err := upsertListItem(userID, listID, mangaID, chapter, statusValue); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	return &mangahubpb.ProgressResponse{
		Message:        "Progress updated",
		MangaId:        mangaID,
		ListName:       listName,
		Status:         statusValue,
		CurrentChapter: int32(chapter),
	}, nil
}

func (s *Server) GetProfile(ctx context.Context, _ *emptypb.Empty) (*mangahubpb.ProfileResponse, error) {
	userID, username, err := extractUser(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	return &mangahubpb.ProfileResponse{Id: userID, Username: username}, nil
}

func (s *Server) GetLibrary(ctx context.Context, req *mangahubpb.GetLibraryRequest) (*mangahubpb.LibraryResponse, error) {
	userID, _, err := extractUser(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	listName := strings.TrimSpace(req.GetListName())
	query := `
		SELECT l.name, li.manga_id, li.current_chapter, li.status, li.updated_at,
			m.title, m.author, m.genres, m.status, m.total_chapters, m.description, m.cover_url
		FROM user_list_items li
		JOIN user_lists l ON l.id = li.list_id
		JOIN manga m ON m.id = li.manga_id
		WHERE li.user_id = ?
	`
	args := []interface{}{userID}
	if listName != "" {
		query += " AND l.name = ?"
		args = append(args, listName)
	}
	query += " ORDER BY li.updated_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	defer rows.Close()

	items := make([]*mangahubpb.LibraryItem, 0)
	for rows.Next() {
		item := &mangahubpb.LibraryItem{Manga: &mangahubpb.Manga{}}
		var genresText string
		if err := rows.Scan(
			&item.ListName,
			&item.Manga.Id,
			&item.CurrentChapter,
			&item.Status,
			&item.UpdatedAt,
			&item.Manga.Title,
			&item.Manga.Author,
			&genresText,
			&item.Manga.Status,
			&item.Manga.TotalChapters,
			&item.Manga.Description,
			&item.Manga.CoverUrl,
		); err != nil {
			return nil, status.Error(codes.Internal, "database error")
		}
		item.Manga.Genres = parseGenres(genresText)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	return &mangahubpb.LibraryResponse{Items: items, Count: int32(len(items))}, nil
}

// clampInt ensures that the provided value is within the specified range, returning a fallback if it's below or a max if it's above.
func clampInt(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

// parseGenres converts a JSON string of genres into a slice of strings, returning an empty slice if the input is empty or invalid.
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

// normalizeListName trims whitespace from the list name and returns a default value if it's empty, ensuring consistent list naming.
func normalizeListName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "reading"
	}
	return name
}

// normalizeStatus trims whitespace from the status value and returns a default value if it's empty, ensuring consistent status naming.
func normalizeStatus(statusValue string) string {
	statusValue = strings.TrimSpace(statusValue)
	switch statusValue {
	case "reading", "completed", "dropped", "plan_to_read":
		return statusValue
	case "":
		return "reading"
	default:
		return "reading"
	}
}

// is it exists in manga table?
func mangaExists(mangaID string) bool {
	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM manga WHERE id = ?)", mangaID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// ensure user list exists for the given user ID and list name, creating it if it doesn't exist
// and returning the list ID or an error if there's a database issue
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

// upsertListItem inserts or updates a user's manga progress in the specified list
// ensuring that the current chapter and status are correctly stored in the database
// and returning an error if there's a database issue
func upsertListItem(userID, listID, mangaID string, currentChapter int, statusValue string) error {
	if currentChapter < 0 {
		currentChapter = 0
	}
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
	`, userID, listID, mangaID, currentChapter, statusValue, time.Now())
	return err
}

// helper function to extract user ID and username from the token in the context metadata
// returning an error if the token is missing or invalid
func extractUser(ctx context.Context) (string, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", fmt.Errorf("missing metadata")
	}

	var token string
	if values := md.Get("authorization"); len(values) > 0 {
		token = strings.TrimSpace(values[0])
	} else if values := md.Get("token"); len(values) > 0 {
		token = strings.TrimSpace(values[0])
	}

	if token == "" {
		return "", "", fmt.Errorf("missing token")
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	userID, username, err := validateToken(token)
	if err != nil {
		return "", "", err
	}
	return userID, username, nil
}

// validateToken creates a JWT token with the given user ID and username
// return: user ID, username
func validateToken(tokenString string) (string, string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", "", fmt.Errorf("JWT_SECRET is not set")
	}
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return "", "", fmt.Errorf("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid token claims")
	}
	userID, _ := claims["sub"].(string)
	username, _ := claims["username"].(string)
	if userID == "" {
		return "", "", fmt.Errorf("missing sub")
	}
	if username == "" {
		username = "user"
	}
	return userID, username, nil
}
