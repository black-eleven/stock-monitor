package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port                  string
	DataDir               string
	Env                   string
	JwtSecret             string
	AdminPassword         string
	ExplicitAdminPassword bool
	DeepSeekAPIKey        string
	DeepSeekModel         string
	DeepSeekBaseURL       string
	LLMCacheTTL           int
	RecommendLimit        int
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	absDataDir, _ := filepath.Abs(dataDir)

	env := os.Getenv("ENV")
	if env == "" {
		env = "mainland"
	}

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

	deepSeekAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	if deepSeekAPIKey == "" {
		log.Printf("[CONFIG] DEEPSEEK_API_KEY not set — LLM recommendation will be unavailable")
	}
	deepSeekModel := os.Getenv("DEEPSEEK_MODEL")
	if deepSeekModel == "" {
		deepSeekModel = "deepseek-chat"
	}
	deepSeekBaseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if deepSeekBaseURL == "" {
		deepSeekBaseURL = "https://api.deepseek.com/v1/chat/completions"
	}
	llmCacheTTL := 30
	if s := os.Getenv("LLM_CACHE_TTL"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			llmCacheTTL = n
		}
	}
	recommendLimit := 8
	if s := os.Getenv("RECOMMEND_LIMIT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 5 && n <= 50 {
			recommendLimit = n
		}
	}

	return &Config{
		Port:                  port,
		DataDir:               absDataDir,
		Env:                   env,
		JwtSecret:             jwtSecret,
		AdminPassword:         adminPassword,
		ExplicitAdminPassword: explicitAdminPassword,
		DeepSeekAPIKey:        deepSeekAPIKey,
		DeepSeekModel:         deepSeekModel,
		DeepSeekBaseURL:       deepSeekBaseURL,
		LLMCacheTTL:           llmCacheTTL,
		RecommendLimit:        recommendLimit,
	}
}

func generateRandomSecret(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
