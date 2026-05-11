package handler

import (
	"net/http"
	"time"

	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type WatchlistHandler struct {
	repo      *repo.WatchlistRepo
	onChanged func()
}

func NewWatchlistHandler(r *repo.WatchlistRepo, onChanged func()) *WatchlistHandler {
	return &WatchlistHandler{repo: r, onChanged: onChanged}
}

func (h *WatchlistHandler) Register(api *gin.RouterGroup) {
	api.GET("/watchlist", h.getAll)
	api.POST("/watchlist", h.add)
	api.DELETE("/watchlist/:symbol", h.remove)
}

func (h *WatchlistHandler) getAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.repo.GetAll(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watchlist"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *WatchlistHandler) add(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Symbol string `json:"symbol"`
		Name   string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and name are required"})
		return
	}
	item := model.WatchlistItem{
		Symbol:  req.Symbol,
		Name:    req.Name,
		AddedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := h.repo.Add(userID, item); err != nil {
		if err == repo.ErrDuplicate {
			c.JSON(http.StatusConflict, gin.H{"error": "Symbol already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add"})
		return
	}
	if h.onChanged != nil {
		h.onChanged()
	}
	c.JSON(http.StatusCreated, item)
}

func (h *WatchlistHandler) remove(c *gin.Context) {
	userID := middleware.GetUserID(c)
	symbol := c.Param("symbol")
	if err := h.repo.Remove(userID, symbol); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
