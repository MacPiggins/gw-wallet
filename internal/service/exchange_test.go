package service

import (
	"context"
	"errors"
	"testing"

	pb "github.com/MacPiggins/gw-proto/go/exchange"
	"github.com/MacPiggins/gw-wallet/internal/cache/inmemory"
	"github.com/MacPiggins/gw-wallet/internal/model"
	"google.golang.org/grpc"
)

type exchangeClientStub struct {
	rates    *pb.ExchangeRatesResponse
	rate     *pb.ExchangeRateResponse
	ratesErr error
	rateErr  error
	request  *pb.CurrencyRequest
}

func (s *exchangeClientStub) GetExchangeRates(context.Context, *pb.Empty, ...grpc.CallOption) (*pb.ExchangeRatesResponse, error) {
	return s.rates, s.ratesErr
}

func (s *exchangeClientStub) GetExchangeRate(_ context.Context, request *pb.CurrencyRequest, _ ...grpc.CallOption) (*pb.ExchangeRateResponse, error) {
	s.request = request
	return s.rate, s.rateErr
}

func newExchangeServiceForTest(client pb.ExchangeServiceClient, cache *inmemory.Cache) *ExchangeService {
	return &ExchangeService{client: client, cache: cache}
}

func TestRatesFetchesAndCaches(t *testing.T) {
	want := map[string]float32{"USD": 1, "EUR": 0.92}
	cache := inmemory.NewCache()
	svc := newExchangeServiceForTest(&exchangeClientStub{rates: &pb.ExchangeRatesResponse{Rates: want}}, cache)

	got, err := svc.Rates(context.Background())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}
	for currency, rate := range want {
		if got[currency] != rate {
			t.Errorf("Rates()[%q] = %v, want %v", currency, got[currency], rate)
		}
		cached, ok := cache.Get("exchange_rate:" + currency)
		if !ok || cached != rate {
			t.Errorf("cached %s rate = %v (present=%v), want %v", currency, cached, ok, rate)
		}
	}
}

func TestRatesReturnsClientError(t *testing.T) {
	wantErr := errors.New("exchange unavailable")
	svc := newExchangeServiceForTest(&exchangeClientStub{ratesErr: wantErr}, inmemory.NewCache())
	got, err := svc.Rates(context.Background())
	if !errors.Is(err, wantErr) || got != nil {
		t.Fatalf("Rates() = (%v, %v), want (nil, %v)", got, err, wantErr)
	}
}

func TestRateUsesCache(t *testing.T) {
	cache := inmemory.NewCache()
	cache.Set("exchange_rate:USDtoEUR", float32(0.91), 0)
	client := &exchangeClientStub{}
	svc := newExchangeServiceForTest(client, cache)
	got, err := svc.Rate(context.Background(), "USD", "EUR")
	if err != nil {
		t.Fatalf("Rate() error = %v", err)
	}
	want := &model.ExchangeRate{From: "USD", To: "EUR", Rate: 0.91}
	if *got != *want {
		t.Errorf("Rate() = %+v, want %+v", got, want)
	}
	if client.request != nil {
		t.Error("Rate() called exchange client for a cached rate")
	}
}

func TestRateFetchesAndCaches(t *testing.T) {
	cache := inmemory.NewCache()
	client := &exchangeClientStub{rate: &pb.ExchangeRateResponse{Rate: 0.87}}
	svc := newExchangeServiceForTest(client, cache)
	got, err := svc.Rate(context.Background(), "GBP", "USD")
	if err != nil {
		t.Fatalf("Rate() error = %v", err)
	}
	if got.From != "GBP" || got.To != "USD" || got.Rate != 0.87 {
		t.Errorf("Rate() = %+v, want GBP/USD at 0.87", got)
	}
	if client.request == nil || client.request.From != "GBP" || client.request.To != "USD" {
		t.Errorf("client request = %+v, want GBP/USD", client.request)
	}
	cached, ok := cache.Get("exchange_rate:GBPtoUSD")
	if !ok || cached != float32(0.87) {
		t.Errorf("cached rate = %v (present=%v), want 0.87", cached, ok)
	}
}

func TestRateReturnsClientError(t *testing.T) {
	wantErr := errors.New("exchange unavailable")
	svc := newExchangeServiceForTest(&exchangeClientStub{rateErr: wantErr}, inmemory.NewCache())
	got, err := svc.Rate(context.Background(), "GBP", "USD")
	if !errors.Is(err, wantErr) || got != nil {
		t.Fatalf("Rate() = (%v, %v), want (nil, %v)", got, err, wantErr)
	}
}
