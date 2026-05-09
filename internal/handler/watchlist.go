package handler

import (
	"net/http"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type WatchlistHandler struct {
	repo *repo.WatchlistRepo
}

func NewWatchlistHandler(r *repo.WatchlistRepo) *WatchlistHandler {
	return &WatchlistHandler{repo: r}
}

func (h *WatchlistHandler) Register(api *gin.RouterGroup) {
	api.GET("/watchlist", h.getAll)
	api.POST("/watchlist", h.add)
	api.DELETE("/watchlist/:symbol", h.remove)
}

func (h *WatchlistHandler) getAll(c *gin.Context) {
	items, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watchlist"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *WatchlistHandler) add(c *gin.Context) {
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
	if err := h.repo.Add(item); err != nil {
		if err == repo.ErrDuplicate {
			c.JSON(http.StatusConflict, gin.H{"error": "Symbol already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *WatchlistHandler) remove(c *gin.Context) {
	symbol := c.Param("symbol")
	if err := h.repo.Remove(symbol); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
