package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/black-eleven/stock-monitor/internal/alert"
	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/black-eleven/stock-monitor/internal/handler"
	"github.com/black-eleven/stock-monitor/internal/llm"
	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/recommend"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/black-eleven/stock-monitor/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	database, err := db.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Repositories
	watchlistRepo := repo.NewWatchlistRepo(database)
	alertRepo := repo.NewAlertRepo(database)
	holdingRepo := repo.NewHoldingRepo(database)
	userRepo := repo.NewUserRepo(database)
	inviteCodeRepo := repo.NewInviteCodeRepo(database)

	// WebSocket Hub
	hub := ws.NewHub(cfg.JwtSecret)
	go hub.Run()

	// Init admin user if first run
	adminID, err := db.InitAdmin(database, cfg.AdminPassword, cfg.ExplicitAdminPassword)
	if err != nil {
		log.Fatalf("Failed to init admin: %v", err)
	}
	if adminID > 0 {
		log.Printf("[MAIN] Initial admin created (id=%d), password printed in config logs above", adminID)
	}

	// EastMoney Client
	emClient := eastmoney.NewClient(cfg.Env)

	// Alert Engine
	alertEngine := alert.NewEngine(alertRepo, hub)

	// HTTP handlers
	watchlistH := handler.NewWatchlistHandler(watchlistRepo, nil)
	alertH := handler.NewAlertHandler(alertRepo)
	holdingH := handler.NewHoldingHandler(holdingRepo)
	quoteH := handler.NewQuoteHandler(emClient)
	klineH := handler.NewKlineHandler(emClient)

	// LLM Client (shared by recommender + strategy)
	var llmClient *llm.Client
	if cfg.DeepSeekAPIKey != "" {
		llmClient = llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekModel, cfg.DeepSeekBaseURL)
	}

	// Recommender
	recommender := recommend.NewRecommender(llmClient, emClient, cfg.LLMCacheTTL, cfg.RecommendLimit)
	recommendH := handler.NewRecommendHandler(recommender, watchlistRepo)

	// Strategy handler
	strategyH := handler.NewStrategyHandler(llmClient, emClient)

	signalRepo := repo.NewSignalRepo(database)
	signalH := handler.NewSignalHandler(signalRepo, hub)

	searchH := handler.NewSearchHandler(emClient)
	fundamentalsH := handler.NewFundamentalsHandler(emClient)
	dashboardH := handler.NewDashboardHandler(hub, watchlistRepo, alertRepo, signalRepo)

	authH := handler.NewAuthHandler(userRepo, inviteCodeRepo, cfg.JwtSecret)
	adminH := handler.NewAdminHandler(inviteCodeRepo)

	r := gin.Default()
	api := r.Group("/api")

	// Public routes — no auth required
	authH.Register(api)

	// Protected routes — JWT required
	authMW := middleware.AuthMiddleware(cfg.JwtSecret)
	auth := api.Group("", authMW)
	watchlistH.Register(auth)
	alertH.Register(auth)
	holdingH.Register(auth)
	quoteH.Register(auth)
	klineH.Register(auth)
	recommendH.Register(auth)
	signalH.Register(auth)
	strategyH.Register(auth)
	searchH.Register(auth)
	fundamentalsH.Register(auth)
	dashboardH.Register(auth)

	// Admin routes
	admin := auth.Group("/admin", middleware.AdminRequired())
	adminH.Register(admin)

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) { hub.ServeWS(c.Writer, c.Request) })

	// Static files (must be last — API and WS routes match first)
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/index.html", "./web/index.html")
	r.StaticFile("/login.html", "./web/login.html")
	r.StaticFile("/admin.html", "./web/admin.html")

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Polling goroutine — fetch quotes every 5s for tracked stocks
	go pollQuotes(emClient, watchlistRepo, hub, alertEngine)

	// Periodic watchlist sync — refresh tracked codes every 30s
	go syncTrackedCodes(emClient, watchlistRepo)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}

func pollQuotes(emClient *eastmoney.Client, watchlistRepo *repo.WatchlistRepo, hub *ws.Hub, alertEngine *alert.Engine) {
	time.Sleep(2 * time.Second)
	for {
		codes := emClient.GetTrackedCodes()
		if len(codes) > 0 {
			quotes, err := emClient.BatchFetchQuotes(codes)
			if err != nil {
				log.Printf("[POLL] Batch fetch error: %v", err)
			} else {
				for _, q := range quotes {
					mq := model.FromEMQuote(*q)
					hub.BroadcastQuote(mq)
					alertEngine.Evaluate(mq)
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func syncTrackedCodes(emClient *eastmoney.Client, watchlistRepo *repo.WatchlistRepo) {
	indexSymbols := handler.IndexSymbols()
	for {
		time.Sleep(30 * time.Second)
		symbols, err := watchlistRepo.GetAllSymbols()
		if err != nil {
			log.Printf("[SYNC] Failed to load watchlist symbols: %v", err)
			continue
		}
		// Always include index symbols in tracked codes
		seen := make(map[string]bool)
		for _, s := range indexSymbols {
			seen[s] = true
		}
		for _, s := range symbols {
			seen[s] = true
		}
		all := make([]string, 0, len(seen))
		for s := range seen {
			all = append(all, s)
		}
		if len(all) > 0 {
			emClient.SetTrackedCodes(all)
		}
	}
}
