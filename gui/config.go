package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type guiConfig struct {
	Mode          int    `json:"mode"`
	ServerAddr    string `json:"serverAddr"`
	BindAddr      string `json:"bindAddr"`
	BlockSize     string `json:"blockSize"`
	Count         string `json:"count"`
	TransferIndex int    `json:"transferIndex"`
	Download      bool   `json:"download"`
	Upload        bool   `json:"upload"`
	OnlyConnect   bool   `json:"onlyConnect"`
}

func configFilePath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "netspeed-gui.json"
	}
	dir := filepath.Dir(exePath)
	return filepath.Join(dir, "netspeed-gui.json")
}

func loadConfig() *guiConfig {
	path := configFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg guiConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Println("load config error:", err)
		return nil
	}
	return &cfg
}

func saveConfig(cfg *guiConfig) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Println("marshal config error:", err)
		return
	}
	path := configFilePath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Println("save config error:", err)
	}
}
