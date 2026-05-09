package main

import (
	"MangaHub/internal/auth"
	"MangaHub/internal/manga"
	"MangaHub/internal/user"
	"MangaHub/pkg/database"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// fail fast if JWT_SECRET is not set, since it's critical for auth security
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	// init database connection
	database.InitDB()
	defer database.Close() // ensure DB connection is closed on application exit
	database.InitSchema()  // create tables and run migrations if needed

	// seed data
	seedPath := os.Getenv("MANGA_SEED_PATH")
	if seedPath == "" {
		seedPath = "data/manga_seed.json"
	}
	if err := database.SeedMangaFromJSON(seedPath); err != nil {
		log.Printf("Seed skipped: %v", err)
	}

	// set up Gin router
	r := setupRouter()

	//run server on port 8080
	log.Println("Starting HTTP server on port :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start HTTP server:", err)
	}
}

// helper function to set up Gin router with CORS and API routes
func setupRouter() *gin.Engine {
	r := gin.Default()

	// Configure CORS
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

	// Manga public routes
	r.GET("/manga", manga.Search)
	r.GET("/manga/:id", manga.GetByID)

	//test route to verify JWT middleware
	apiGroup := r.Group("/api")
	apiGroup.Use(auth.JWTMiddleware())
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

	// User library routes (protected)
	userGroup := r.Group("/users")
	userGroup.Use(auth.JWTMiddleware())
	{
		userGroup.POST("/library", user.AddToLibrary)
		userGroup.GET("/library", user.GetLibrary)
		userGroup.PUT("/progress", user.UpdateProgress)
	}
	return r
}
