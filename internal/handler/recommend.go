package handler

import (
	"net/http"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/recommend"
	"github.com/gin-gonic/gin"
)

type RecommendHandler struct {
	recommender *recommend.Recommender
}

func NewRecommendHandler(r *recommend.Recommender) *RecommendHandler {
	return &RecommendHandler{recommender: r}
}

func (h *RecommendHandler) Register(api *gin.RouterGroup) {
	api.POST("/recommendations", h.recommend)
}

func (h *RecommendHandler) recommend(c *gin.Context) {
	var req model.RecommendReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Industry) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "industry is required"})
		return
	}

	recs, err := h.recommender.Search(strings.TrimSpace(req.Industry))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch recommendations: " + err.Error()})
		return
	}

	if recs == nil {
		recs = []model.Recommendation{}
	}

	c.JSON(http.StatusOK, model.RecommendResp{Recommendations: recs})
}
