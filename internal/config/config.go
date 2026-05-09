package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port     string
	QosKey   string
	DataDir  string
	QosWsUrl string
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
	return &Config{
		Port:     port,
		QosKey:   qosKey,
		DataDir:  absDataDir,
		QosWsUrl: qosWsUrl,
	}
}
