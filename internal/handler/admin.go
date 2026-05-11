package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	inviteCodeRepo *repo.InviteCodeRepo
}

func NewAdminHandler(inviteCodeRepo *repo.InviteCodeRepo) *AdminHandler {
	return &AdminHandler{inviteCodeRepo: inviteCodeRepo}
}

func (h *AdminHandler) Register(admin *gin.RouterGroup) {
	admin.POST("/invite-codes", h.createCodes)
	admin.GET("/invite-codes", h.listCodes)
	admin.PUT("/invite-codes/:id", h.updateCode)
}

func (h *AdminHandler) createCodes(c *gin.Context) {
	var req model.CreateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.MaxUses = 1
		req.Count = 1
	}
	if req.Count <= 0 {
		req.Count = 1
	}

	userID, _ := c.Get("user_id")
	now := time.Now().UTC().Format(time.RFC3339)
	var codes []model.InviteCode

	for i := 0; i < req.Count; i++ {
		b := make([]byte, 8)
		rand.Read(b)
		code := model.InviteCode{
			Code:      hex.EncodeToString(b),
			MaxUses:   req.MaxUses,
			CreatedBy: userID.(int),
			CreatedAt: now,
			IsActive:  true,
		}
		id, err := h.inviteCodeRepo.Create(code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create code"})
			return
		}
		code.ID = id
		codes = append(codes, code)
	}

	c.JSON(http.StatusCreated, codes)
}

func (h *AdminHandler) listCodes(c *gin.Context) {
	userID, _ := c.Get("user_id")
	codes, err := h.inviteCodeRepo.ListByCreator(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list codes"})
		return
	}
	c.JSON(http.StatusOK, codes)
}

func (h *AdminHandler) updateCode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var req struct {
		IsActive *bool `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	if err := h.inviteCodeRepo.SetActive(id, active); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
