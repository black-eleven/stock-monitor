package handler

import (
	"net/http"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	client *eastmoney.Client
}

func NewSearchHandler(client *eastmoney.Client) *SearchHandler {
	return &SearchHandler{client: client}
}

func (h *SearchHandler) Register(api *gin.RouterGroup) {
	api.GET("/search", h.search)
}

func (h *SearchHandler) search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing query"})
		return
	}
	results, err := h.client.SearchStocks(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}
