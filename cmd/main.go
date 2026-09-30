package main

import (
	"log/slog"
	"os"

	"github.com/MacPiggins/gw-wallet/internal/app"
	"github.com/MacPiggins/gw-wallet/internal/config"
	"github.com/MacPiggins/gw-wallet/internal/logging"
)

// @title gw-wallet
// @version 1.0
// @description Command-line entry point for the gw-wallet application.
// @BasePath /
func main() {
	h := &logging.ContextHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)}
	slog.SetDefault(slog.New(h))

	conf, err := config.Load("config.env")
	if err != nil {
		slog.Error("error while loading config", slog.Any("error", err))
		return
	}
	app.Run(conf)
}
