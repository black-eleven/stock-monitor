package handler

import (
	"net/http"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/gin-gonic/gin"
)

type FundamentalsHandler struct {
	client eastmoney.QuoteClient
}

func NewFundamentalsHandler(client eastmoney.QuoteClient) *FundamentalsHandler {
	return &FundamentalsHandler{client: client}
}

func (h *FundamentalsHandler) Register(api *gin.RouterGroup) {
	api.GET("/fundamentals/:symbol", h.getFundamentals)
}

func (h *FundamentalsHandler) getFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	data, err := h.client.FetchFundamentalsCached(symbol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch fundamentals"})
		return
	}
	c.JSON(http.StatusOK, data)
}
