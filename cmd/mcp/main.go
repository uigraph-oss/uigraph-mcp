package main

import (
	"log/slog"
	"os"

	"github.com/uigraph/mcp/internal/config"
	"github.com/uigraph/mcp/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	_ = cfg // mcp server wired in Task B4
	slog.Info("uigraph-mcp starting", "apiURL", cfg.UIGraphAPIURL)
	if err := server.Run(cfg.Port, nil); err != nil {
		slog.Error("run error", "err", err)
		os.Exit(1)
	}
}
