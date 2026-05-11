package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Port           string
	QosKey         string
	DataDir        string
	QosWsUrl       string
	JwtSecret             string
	AdminPassword          string
	ExplicitAdminPassword  bool
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

	return &Config{
		Port:          port,
		QosKey:        qosKey,
		DataDir:       absDataDir,
		QosWsUrl:      qosWsUrl,
		JwtSecret:     jwtSecret,
		AdminPassword:          adminPassword,
		ExplicitAdminPassword:  explicitAdminPassword,
	}
}

func generateRandomSecret(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
