package main

import (
	"MangaHub/internal/auth"
	"MangaHub/pkg/database"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const schema = `
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
	description TEXT
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
`

func main() {
	// fail fast if JWT_SECRET is not set, since it's critical for auth security
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	// Initialize database connection
	database.InitDB()
	defer database.Close()         // Ensure DB connection is closed on application exit
	database.ExecuteSchema(schema) // Create tables if they don't exist

	//build gin engine with logger and recovery middleware
	r := gin.Default() // Default() includes Logger and Recovery middleware

	//configure CROS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins for development
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Register API routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"service":   "mangahub-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})
	// Auth routes (registration and login)
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", auth.Register)
		authGroup.POST("/login", auth.Login)
	}

	//test route to verify JWT middleware
	apiGroup := r.Group("/api")
	apiGroup.Use(auth.JWTMiddleware()) // protect all /api routes with JWT auth
	{
		apiGroup.GET("/profile", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			username, _ := c.Get("username")
			c.JSON(http.StatusOK, gin.H{
				"id":       userID,
				"username": username,
			})
		})
	}

	//run server on port 8080
	log.Println("Starting HTTP server on port :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start HTTP server:", err)
	}
}
