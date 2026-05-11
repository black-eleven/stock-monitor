package handler

import (
	"net/http"
	"time"

	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type HoldingHandler struct {
	repo *repo.HoldingRepo
}

func NewHoldingHandler(r *repo.HoldingRepo) *HoldingHandler {
	return &HoldingHandler{repo: r}
}

func (h *HoldingHandler) Register(api *gin.RouterGroup) {
	api.GET("/holdings", h.getAll)
	api.POST("/holdings", h.add)
	api.PUT("/holdings/:symbol", h.update)
	api.DELETE("/holdings/:symbol", h.remove)
}

func (h *HoldingHandler) getAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.repo.GetAll(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch holdings"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *HoldingHandler) add(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Symbol  string  `json:"symbol"`
		Name    string  `json:"name"`
		Shares  float64 `json:"shares"`
		AvgCost float64 `json:"avgCost"`
		BuyDate string  `json:"buyDate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Shares == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol, shares, and avgCost are required"})
		return
	}
	if req.BuyDate == "" {
		req.BuyDate = time.Now().Format("2006-01-02")
	}
	item := model.Holding{
		Symbol:  req.Symbol,
		Name:    req.Name,
		Shares:  req.Shares,
		AvgCost: req.AvgCost,
		BuyDate: req.BuyDate,
	}
	if err := h.repo.Add(userID, item); err != nil {
		if err == repo.ErrDuplicate {
			c.JSON(http.StatusConflict, gin.H{"error": "Holding already exists for this symbol"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *HoldingHandler) update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	symbol := c.Param("symbol")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}
	err := h.repo.Update(userID, symbol, func(h *model.Holding) {
		if v, ok := req["shares"]; ok {
			h.Shares = v.(float64)
		}
		if v, ok := req["avgCost"]; ok {
			h.AvgCost = v.(float64)
		}
		if v, ok := req["buyDate"]; ok {
			h.BuyDate = v.(string)
		}
		if v, ok := req["name"]; ok {
			h.Name = v.(string)
		}
	})
	if err == repo.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Holding not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *HoldingHandler) remove(c *gin.Context) {
	userID := middleware.GetUserID(c)
	symbol := c.Param("symbol")
	if err := h.repo.Remove(userID, symbol); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Holding not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
