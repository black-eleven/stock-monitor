package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	repo *repo.AlertRepo
}

func NewAlertHandler(r *repo.AlertRepo) *AlertHandler {
	return &AlertHandler{repo: r}
}

func (h *AlertHandler) Register(api *gin.RouterGroup) {
	api.GET("/alerts", h.getAll)
	api.POST("/alerts", h.add)
	api.PUT("/alerts/:id", h.update)
	api.DELETE("/alerts/:id", h.remove)
}

func (h *AlertHandler) getAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	rules, err := h.repo.GetAll(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *AlertHandler) add(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Symbol string  `json:"symbol"`
		Type   string  `json:"type"`
		Value  float64 `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol, type, and value are required"})
		return
	}
	if req.Type != "above" && req.Type != "below" && req.Type != "change_pct" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be above, below, or change_pct"})
		return
	}
	rule := model.AlertRule{
		Symbol:    req.Symbol,
		Type:      req.Type,
		Value:     req.Value,
		Enabled:   true,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	id, err := h.repo.Add(userID, rule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add alert"})
		return
	}
	rule.ID = id
	c.JSON(http.StatusCreated, rule)
}

func (h *AlertHandler) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}
	err = h.repo.Update(id, func(a *model.AlertRule) {
		if v, ok := req["type"]; ok {
			a.Type = v.(string)
		}
		if v, ok := req["value"]; ok {
			a.Value = v.(float64)
		}
		if v, ok := req["enabled"]; ok {
			a.Enabled = v.(bool)
		}
	})
	if err == repo.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AlertHandler) remove(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := h.repo.Remove(id); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
