package app

import (
	"context"
	"log/slog"

	"github.com/MacPiggins/gw-wallet/internal/broker"
	"github.com/MacPiggins/gw-wallet/internal/cache/inmemory"
	"github.com/MacPiggins/gw-wallet/internal/config"
	"github.com/MacPiggins/gw-wallet/internal/server"
	"github.com/MacPiggins/gw-wallet/internal/service"
	"github.com/MacPiggins/gw-wallet/internal/storage/postgres"
)

func Run(conf *config.Config) error {
	ctx := context.Background()
	storage, err := postgres.New(ctx, conf.StorageConnstr)
	if err != nil {
		slog.Error("error while creating storage", slog.Any("error", err))
		return err
	}
	err = storage.AutoMigrate(ctx)
	if err != nil {
		slog.Error("error while migrating storage", slog.Any("error", err))
		return err
	}

	cache := inmemory.NewCache()

	broker, err := broker.NewNats(ctx, conf)
	if err != nil {
		slog.Error("error while creating broker", slog.Any("error", err))
		return err
	}
	defer broker.Close()

	authService := service.NewAuthService(storage, []byte(conf.AuthSecret))
	exchangeService, err := service.NewExchangeService(conf, cache)
	if err != nil {
		slog.Error("error while creating exchange service", slog.Any("error", err))
		return err
	}
	walletService := service.NewWalletService(storage, broker)

	handler := server.NewHandler(authService, exchangeService, walletService, broker)
	server := server.NewServer(handler)

	err = server.Run(conf.ServerAddr)
	if err != nil {
		slog.Error("error while running server", slog.Any("error", err))
		return err
	}
	return nil
}
