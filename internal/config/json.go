package config

import (
	"encoding/json"
	"os"
)

type ServerFileConfig struct {
	Address       string `json:"address"`
	GRPCAddress   string `json:"grpc_address"`
	Restore       *bool  `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	TrustedSubnet string `json:"trusted_subnet"`
	Key           string `json:"key"`
	AuditFile     string `json:"audit_file"`
	AuditURL      string `json:"audit_url"`
}

type AgentFileConfig struct {
	Address        string `json:"address"`
	GRPCAddress    string `json:"grpc_address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
	Key            string `json:"key"`
	RateLimit      *int   `json:"rate_limit"`
}

func LoadServerFileConfig(path string) (ServerFileConfig, error) {
	var cfg ServerFileConfig
	err := loadJSON(path, &cfg)
	return cfg, err
}

func LoadAgentFileConfig(path string) (AgentFileConfig, error) {
	var cfg AgentFileConfig
	err := loadJSON(path, &cfg)
	return cfg, err
}

func loadJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}
