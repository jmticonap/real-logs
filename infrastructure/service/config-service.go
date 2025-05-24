package service

import (
	"encoding/json"
	"os"

	"github.com/jmticonap/real-logs/domain"
)

func LoadConfig(path string) (*domain.Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg domain.Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
