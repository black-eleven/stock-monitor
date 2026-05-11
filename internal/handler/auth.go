package handler

import (
	"net/http"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo       *repo.UserRepo
	inviteCodeRepo *repo.InviteCodeRepo
	jwtSecret      string
}

func NewAuthHandler(userRepo *repo.UserRepo, inviteCodeRepo *repo.InviteCodeRepo, jwtSecret string) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, inviteCodeRepo: inviteCodeRepo, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Register(api *gin.RouterGroup) {
	api.POST("/auth/register", h.register)
	api.POST("/auth/login", h.login)
}

func (h *AuthHandler) register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username, password, and inviteCode are required"})
		return
	}
	if len(req.Username) < 2 || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username at least 2 chars, password at least 6 chars"})
		return
	}

	// Validate and consume invite code
	_, err := h.inviteCodeRepo.GetByCode(req.InviteCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invite code"})
		return
	}

	if err := h.inviteCodeRepo.IncrementUsed(req.InviteCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invite code is expired or exhausted"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	u := model.User{
		Username:  req.Username,
		Password:  string(hash),
		Role:      "user",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	id, err := h.userRepo.Create(u)
	if err != nil {
		if err == repo.ErrDuplicate {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	token, err := h.generateToken(id, u.Username, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	u.ID = id
	c.JSON(http.StatusCreated, model.LoginResponse{Token: token, User: u})
}

func (h *AuthHandler) login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	u, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	token, err := h.generateToken(u.ID, u.Username, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, model.LoginResponse{Token: token, User: *u})
}

func (h *AuthHandler) generateToken(userID int, username, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
