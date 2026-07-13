package main

import (
	"log/slog"
	"os"

	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/config"
	mcphandler "github.com/uigraph/mcp/internal/mcp"
	"github.com/uigraph/mcp/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	client := apiclient.New(cfg.UIGraphAPIURL, cfg.UIGraphGatewayURL)
	handler := mcphandler.New(cfg, client)

	slog.Info("uigraph-mcp starting", "port", cfg.Port)
	if err := server.Run(cfg.Port, handler); err != nil {
		slog.Error("run error", "err", err)
		os.Exit(1)
	}
}
