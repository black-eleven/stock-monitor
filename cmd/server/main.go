package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/black-eleven/stock-monitor/internal/alert"
	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/handler"
	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/qos"
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

	// QOS Client
	qosClient := qos.NewClient(cfg.QosWsUrl)

	// Alert Engine
	alertEngine := alert.NewEngine(alertRepo, hub)

	// Wire QOS callbacks
	qosClient.OnQuote = func(q qos.Quote) {
		mq := model.FromQosQuote(q)
		hub.BroadcastQuote(mq)
		alertEngine.Evaluate(mq)
	}

	// HTTP handlers
	watchlistH := handler.NewWatchlistHandler(watchlistRepo, nil)
	alertH := handler.NewAlertHandler(alertRepo)
	holdingH := handler.NewHoldingHandler(holdingRepo)
	quoteH := handler.NewQuoteHandler(qosClient)
	klineH := handler.NewKlineHandler(qosClient)

	// Recommender
	newsapiClient := recommend.NewNewsAPIClient(cfg.NewsAPIKey)
	recommender := recommend.NewRecommender(newsapiClient, qosClient, cfg.NewsAPIDays, cfg.NewsAPIPageSize, cfg.NewsAPILanguages, cfg.RecommendCandidates, cfg.RecommendLimit)
	recommendH := handler.NewRecommendHandler(recommender)

	signalRepo := repo.NewSignalRepo(database)
	signalH := handler.NewSignalHandler(signalRepo, hub)

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

	// Connect QOS after server is ready
	go qosClient.Connect()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	qosClient.Close()
}
