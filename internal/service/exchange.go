package service

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/MacPiggins/gw-proto/go/exchange"
	"github.com/MacPiggins/gw-wallet/internal/cache/inmemory"
	"github.com/MacPiggins/gw-wallet/internal/config"
	"github.com/MacPiggins/gw-wallet/internal/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ExchangeService struct {
	conn   *grpc.ClientConn
	client pb.ExchangeServiceClient
	cache  *inmemory.Cache
}

func NewExchangeService(conf *config.Config, cache *inmemory.Cache) (*ExchangeService, error) {
	conn, err := grpc.NewClient(conf.ExchangeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to exchange service", slog.Any("error", err))
		return nil, err
	}	
	exchangeClient := pb.NewExchangeServiceClient(conn)
	return &ExchangeService{conn: conn, client: exchangeClient, cache: cache}, nil
}

func (svc *ExchangeService) CloseConn() {
	svc.conn.Close()
}

func (svc *ExchangeService) Rates(ctx context.Context) (map[string]float32, error) {
	response, err := svc.client.GetExchangeRates(ctx, &pb.Empty{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to retrieve exchange rates", slog.Any("error", err))
		return nil, err
	}
	for currency, rate := range response.Rates {
		svc.cache.Set("exchange_rate:"+currency, rate, time.Minute)
	}
	return response.Rates, nil
}

func (svc *ExchangeService) Rate(ctx context.Context, from, to string) (*model.ExchangeRate, error) {
	cacheKey := "exchange_rate:" + from + "to" + to
	if cachedRate, ok := svc.cache.Get(cacheKey); ok {
		if rate, ok := cachedRate.(float32); ok {
			return &model.ExchangeRate{From: from, To: to, Rate: rate}, nil
		}
	}

	response, err := svc.client.GetExchangeRate(ctx, &pb.CurrencyRequest{From: from, To: to})
	if err != nil {
		slog.ErrorContext(ctx, "failed to retrieve exchange rate", slog.Any("error", err))
		return nil, err
	}
	svc.cache.Set(cacheKey, response.Rate, time.Minute)
	rate := &model.ExchangeRate{From: from, To: to, Rate: response.Rate}
	return rate, nil
}
