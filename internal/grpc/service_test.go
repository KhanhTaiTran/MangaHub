package grpc

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"MangaHub/pkg/database"
	"MangaHub/proto/mangahubpb"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024 // constant for bufconn buffer size

// use to test search and progress update in one test to avoid multiple setup/teardown of the server and database
func TestSearchAndUpdateProgress(t *testing.T) {
	conn, token, mangaID, cleanup := setupTestServer(t)
	defer cleanup()

	client := mangahubpb.NewMangaServiceClient(conn) // call MangaService

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // set timeout to prevent hanging tests
	defer cancel()

	searchResp, err := client.SearchManga(ctx, &mangahubpb.SearchRequest{Query: "sample"})
	if err != nil {
		t.Fatalf("SearchManga error: %v", err)
	}
	if searchResp.Count == 0 {
		t.Fatalf("expected search results")
	}

	ctx = withAuth(ctx, token)
	progressResp, err := client.UpdateProgress(ctx, &mangahubpb.ProgressRequest{
		MangaId:        mangaID,
		ListName:       "reading",
		Status:         "reading",
		CurrentChapter: 3,
	})
	if err != nil {
		t.Fatalf("UpdateProgress error: %v", err)
	}
	if progressResp.GetCurrentChapter() != 3 {
		t.Fatalf("unexpected chapter: %d", progressResp.GetCurrentChapter())
	}

	var chapter int
	var status string
	row := database.DB.QueryRow(
		"SELECT current_chapter, status FROM user_list_items WHERE user_id = ? AND manga_id = ?",
		extractUserIDFromToken(t, token),
		mangaID,
	)
	if err := row.Scan(&chapter, &status); err != nil {
		t.Fatalf("failed to read progress: %v", err)
	}
	if chapter != 3 || status != "reading" {
		t.Fatalf("unexpected db values: chapter=%d status=%s", chapter, status)
	}
}

// setupTestServer initializes an in-memory database, starts a gRPC server with the MangaService and UserService
// returns a client connection, a JWT token for authentication, a manga ID for testing, and a cleanup function to close resources after tests.
func setupTestServer(t *testing.T) (*grpc.ClientConn, string, string, func()) {
	t.Helper()

	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("DB_PATH", ":memory:")

	database.InitDB()
	database.InitSchema()

	userID := uuid.New().String()
	username := "tester"
	_, err := database.DB.Exec(
		"INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)",
		userID,
		username,
		"hash",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	mangaID := "manga-1"
	genres, _ := json.Marshal([]string{"action"})
	_, err = database.DB.Exec(
		"INSERT INTO manga (id, title, author, genres, status, total_chapters, description, cover_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		mangaID,
		"Sample Manga",
		"Author",
		string(genres),
		"ongoing",
		100,
		"Sample",
		"",
	)
	if err != nil {
		t.Fatalf("insert manga: %v", err)
	}

	lis := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	service := NewServer()
	mangahubpb.RegisterMangaServiceServer(server, service)
	mangahubpb.RegisterUserServiceServer(server, service)

	go func() {
		_ = server.Serve(lis)
	}()

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}

	cleanup := func() {
		conn.Close()
		server.Stop()
		lis.Close()
		_ = database.Close()
	}

	token := makeToken(userID, username)
	return conn, token, mangaID, cleanup
}

// makeToken creates a JWT token with the given user ID and username, signed with the secret from environment variables, and an expiration time of 1 hour.
func makeToken(userID, username string) string {
	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	value, _ := token.SignedString([]byte(secret))
	return value
}

// withAuth adds the given JWT token to the context metadata for authentication in gRPC calls.
func withAuth(ctx context.Context, token string) context.Context {
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	return metadata.NewOutgoingContext(ctx, md)
}

// extractUserIDFromToken parses the JWT token string and extracts the user ID from the "sub" claim
// returning it or failing the test if the token is invalid.
func extractUserIDFromToken(t *testing.T, tokenString string) string {
	secret := os.Getenv("JWT_SECRET")
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("invalid token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("invalid claims")
	}
	userID, _ := claims["sub"].(string)
	return userID
}
