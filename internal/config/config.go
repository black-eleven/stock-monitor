package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	QosKey         string
	DataDir        string
	QosWsUrl       string
	JwtSecret             string
	AdminPassword          string
	ExplicitAdminPassword  bool
	NewsAPIKey          string
	NewsAPIDays         int
	NewsAPIPageSize     int
	NewsAPILanguages    []string
	RecommendCandidates int
	RecommendLimit      int
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	qosKey := os.Getenv("QOS_KEY")
	qosWsUrl := "wss://api.qos.hk/ws"
	if qosKey != "" {
		qosWsUrl += "?key=" + qosKey
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	absDataDir, _ := filepath.Abs(dataDir)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = generateRandomSecret(32)
		log.Printf("[CONFIG] JWT_SECRET not set, generated random secret (first 8 chars): %s...", jwtSecret[:8])
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	explicitAdminPassword := adminPassword != ""
	if adminPassword == "" {
		adminPassword = generateRandomSecret(16)
		log.Printf("[CONFIG] ADMIN_PASSWORD not set, generated: %s", adminPassword)
	}

	newsAPIKey := os.Getenv("NEWSAPI_KEY")
	if newsAPIKey == "" {
		log.Printf("[CONFIG] NEWSAPI_KEY not set — recommendation feature will be unavailable")
	}
	newsAPILanguages := []string{"en", "zh"}
	if s := os.Getenv("NEWSAPI_LANGUAGES"); s != "" {
		newsAPILanguages = strings.Split(s, ",")
	}
	newsAPIDays := 7
	if s := os.Getenv("NEWSAPI_DAYS"); s != "" {
		if d, err := strconv.Atoi(s); err == nil && d > 0 {
			newsAPIDays = d
		}
	}
	newsAPIPageSize := 50
	if s := os.Getenv("NEWSAPI_PAGE_SIZE"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			newsAPIPageSize = n
		}
	}

	recommendCandidates := 20
	if s := os.Getenv("RECOMMEND_CANDIDATES"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 5 && n <= 50 {
			recommendCandidates = n
		}
	}
	recommendLimit := 15
	if s := os.Getenv("RECOMMEND_LIMIT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 5 && n <= 50 {
			recommendLimit = n
		}
	}

	return &Config{
		Port:          port,
		QosKey:        qosKey,
		DataDir:       absDataDir,
		QosWsUrl:      qosWsUrl,
		JwtSecret:     jwtSecret,
		AdminPassword:          adminPassword,
		ExplicitAdminPassword:  explicitAdminPassword,
		NewsAPIKey:          newsAPIKey,
		NewsAPIDays:         newsAPIDays,
		NewsAPIPageSize:     newsAPIPageSize,
		NewsAPILanguages:    newsAPILanguages,
		RecommendCandidates: recommendCandidates,
		RecommendLimit:      recommendLimit,
	}
}

func generateRandomSecret(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
