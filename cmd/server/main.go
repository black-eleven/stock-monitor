package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/handler"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

func main() {
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

	// Handlers
	watchlistH := handler.NewWatchlistHandler(watchlistRepo)
	alertH := handler.NewAlertHandler(alertRepo)
	holdingH := handler.NewHoldingHandler(holdingRepo)

	r := gin.Default()
	api := r.Group("/api")
	watchlistH.Register(api)
	alertH.Register(api)
	holdingH.Register(api)

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}
