package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	JWTSecret         string
	JWTExpirationHrs  int
	AdminUsername     string
	AdminPassword     string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(secret string, expirationHours int, adminUser, adminPass string) *AuthHandler {
	return &AuthHandler{
		JWTSecret:        secret,
		JWTExpirationHrs: expirationHours,
		AdminUsername:    adminUser,
		AdminPassword:    adminPass,
	}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a user and returns a JWT token.
// Credentials are configurable via ADMIN_USERNAME/ADMIN_PASSWORD env vars.
// TODO: Replace with main backend SSO delegation in production.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "username and password are required")
		return
	}

	// Dev-mode credential check.
	// TODO: Replace with main backend SSO delegation in production.
	if req.Username != h.AdminUsername || req.Password != h.AdminPassword {
		Error(c, http.StatusUnauthorized, "invalid username or password")
		return
	}

	expHours := h.JWTExpirationHrs
	if expHours <= 0 {
		expHours = 24
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      "1",
		"username": req.Username,
		"iat":      now.Unix(),
		"exp":      now.Add(time.Duration(expHours) * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		Error(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	Success(c, gin.H{
		"token":    tokenString,
		"username": req.Username,
		"roles":    []string{"admin"},
	})
}

// GetUserInfo returns the current user's info (extracted from JWT claims).
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		Error(c, http.StatusUnauthorized, "missing user context")
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		Error(c, http.StatusUnauthorized, "missing user context")
		return
	}

	Success(c, gin.H{
		"user": gin.H{
			"userId":   userID,
			"userName": username,
			"avatar":   "",
		},
		"roles":       []string{"admin"},
		"permissions": []string{"*:*:*"},
	})
}

// Logout is a no-op on the server side (JWT is stateless).
// The client should discard the token.
func (h *AuthHandler) Logout(c *gin.Context) {
	Success(c, gin.H{"message": "logged out"})
}
