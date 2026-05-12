package auth

import (
	"MangaHub/pkg/database"
	"MangaHub/pkg/models"
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Handler for user registration
func Register(c *gin.Context) {
	var req RegisterRequest
	// Validate JSON body and binding rules
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	//check duplicate username
	var existUsername bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", req.Username).Scan(&existUsername)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if existUsername {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	//hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}

	// Insert new user into database
	user := models.Users{
		ID:           uuid.New().String(),
		Username:     req.Username,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
	}
	_, err = database.DB.Exec("INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)", user.ID, user.Username, user.PasswordHash, user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting user into database"})
		return
	}
	// Return created user info (without password hash)
	c.JSON(http.StatusCreated, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"created_at": user.CreatedAt,
	})
}

// helper function to hash password using bcrypt
func HashPassword(password string) (string, error) {
	// Use bcrypt to hash the password with a default cost
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// Login handler to authenticate user
func Login(c *gin.Context) {
	var req LoginRequest
	// Validate JSON body and binding rules
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Fetch user from database by username
	var user models.Users
	err := database.DB.QueryRow("SELECT id, username, password_hash, created_at FROM users WHERE username = ?", req.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Verify password
	if !ComparePassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	//generate JWT token
	secret := os.Getenv("JWT_SECRET")
	expirationTime := time.Now().Add(24 * time.Hour) // token valid for 24 hours
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"exp":      expirationTime.Unix(),
	})
	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      tokenString,
		"expires_at": expirationTime,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"created_at": user.CreatedAt,
		},
	})
}

// helper function to compare hashed password with plain text password
func ComparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
