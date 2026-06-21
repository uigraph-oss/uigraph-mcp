package config

import (
	"fmt"
	"os"
)

type Config struct {
	UIGraphAPIURL   string
	Port            string
	MCPServerName   string
	MCPServerVersion string
}

func Load() (*Config, error) {
	apiURL := os.Getenv("UIGRAPH_API_URL")
	if apiURL == "" {
		return nil, fmt.Errorf("UIGRAPH_API_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	name := os.Getenv("MCP_SERVER_NAME")
	if name == "" {
		name = "uigraph-mcp"
	}
	version := os.Getenv("MCP_SERVER_VERSION")
	if version == "" {
		version = "0.1.0"
	}
	return &Config{
		UIGraphAPIURL:    apiURL,
		Port:             port,
		MCPServerName:    name,
		MCPServerVersion: version,
	}, nil
}
