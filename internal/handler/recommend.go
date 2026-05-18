package handler

import (
	"net/http"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/recommend"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type RecommendHandler struct {
	recommender   *recommend.Recommender
	watchlistRepo *repo.WatchlistRepo
}

func NewRecommendHandler(r *recommend.Recommender, w *repo.WatchlistRepo) *RecommendHandler {
	return &RecommendHandler{recommender: r, watchlistRepo: w}
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

	// Build exclusion list from user's watchlist
	var exclude []string
	userID := middleware.GetUserID(c)
	if items, err := h.watchlistRepo.GetAll(userID); err == nil {
		for _, item := range items {
			exclude = append(exclude, item.Symbol)
		}
	}

	recs, err := h.recommender.Search(strings.TrimSpace(req.Industry), exclude)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch recommendations: " + err.Error()})
		return
	}

	if recs == nil {
		recs = []model.Recommendation{}
	}

	c.JSON(http.StatusOK, model.RecommendResp{Recommendations: recs})
}
