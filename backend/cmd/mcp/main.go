package main

import (
	"gridea-pro/backend/internal/mcp"
	"log/slog"
	"os"
)

// Version 由构建配方注入：-ldflags "-X main.Version=<版本号>"。
// 之前配方一直带着这个注入位，但本包没声明该变量，注入被链接器静默丢弃，
// go version -m 里从未真正出现过版本戳（2026-11-18 huigg.6 修复）。
var Version = "dev"

func main() {
	// 真实引用 Version：未引用的包级变量会被链接器死代码消除，
	// 导致 -X main.Version 注入落空（2026-11-18 huigg.6 排查结论）。
	slog.Info("gridea-pro mcp starting", "version", Version)

	if _, err := os.Stat(mcp.GetAppDir()); os.IsNotExist(err) {
		slog.Error("SOURCE_DIR not found", "path", mcp.GetAppDir())
		slog.Error("Please set SOURCE_DIR environment variable to your Gridea Pro data directory.")
		os.Exit(1)
	}

	server := mcp.NewServer()
	if err := server.Start(); err != nil {
		slog.Error("Error starting MCP server", "error", err)
		os.Exit(1)
	}
}
