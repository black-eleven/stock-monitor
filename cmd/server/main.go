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
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/qos"
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

	// WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

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
	watchlistH := handler.NewWatchlistHandler(watchlistRepo)
	alertH := handler.NewAlertHandler(alertRepo)
	holdingH := handler.NewHoldingHandler(holdingRepo)
	quoteH := handler.NewQuoteHandler(qosClient)
	klineH := handler.NewKlineHandler(qosClient)

	r := gin.Default()
	api := r.Group("/api")
	watchlistH.Register(api)
	alertH.Register(api)
	holdingH.Register(api)
	quoteH.Register(api)
	klineH.Register(api)

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) { hub.ServeWS(c.Writer, c.Request) })

	// Static files (must be last — API and WS routes match first)
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/index.html", "./web/index.html")

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
