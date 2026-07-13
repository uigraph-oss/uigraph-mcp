package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	UIGraphAPIURL     string
	UIGraphGatewayURL string
	FrontendURL       string
	MCPPublicURL      string
	Port              string
	MCPServerName     string
	MCPServerVersion  string
}

func Load() (*Config, error) {
	apiURL := os.Getenv("UIGRAPH_API_URL")
	if apiURL == "" {
		return nil, fmt.Errorf("UIGRAPH_API_URL is required")
	}
	gatewayURL := os.Getenv("UIGRAPH_GATEWAY_URL")
	if gatewayURL == "" {
		return nil, fmt.Errorf("UIGRAPH_GATEWAY_URL is required")
	}
	frontendURL := os.Getenv("UIGRAPH_FRONTEND_URL")
	if frontendURL == "" {
		return nil, fmt.Errorf("UIGRAPH_FRONTEND_URL is required")
	}
	publicURL := os.Getenv("UIGRAPH_MCP_PUBLIC_URL")
	if publicURL == "" {
		return nil, fmt.Errorf("UIGRAPH_MCP_PUBLIC_URL is required")
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
		UIGraphAPIURL:     apiURL,
		UIGraphGatewayURL: strings.TrimRight(gatewayURL, "/"),
		FrontendURL:       strings.TrimRight(frontendURL, "/"),
		MCPPublicURL:      strings.TrimRight(publicURL, "/"),
		Port:              port,
		MCPServerName:     name,
		MCPServerVersion:  version,
	}, nil
}
