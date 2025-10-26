package pkg

import (
	"encoding/json"
	"os"

	"github.com/tik-choco-lab/mistnet-signaling/pkg/logger"
)

type MistConfig struct {
	GlobalNode struct {
		Enable bool
		Port   int
	}
}

const configFile = "config.json"

func LoadConfig() (*MistConfig, error) {
	logger.Info("LoadConfig")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return createDefaultConfig()
	}

	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config MistConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func createDefaultConfig() (*MistConfig, error) {
	config := MistConfig{
		GlobalNode: struct {
			Enable bool
			Port   int
		}{
			Enable: true,
			Port:   8080,
		},
	}

	json, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(configFile, json, 0644)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
