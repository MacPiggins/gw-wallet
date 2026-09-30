package service

import (
	"context"
	"errors"
	"testing"

	"github.com/MacPiggins/gw-wallet/internal/model"
)

type walletStorageStub struct {
	depositWallet  *model.Wallet
	withdrawWallet *model.Wallet
	balanceWallet  *model.Wallet
	createWallet   *model.Wallet
	exchangeWallet *model.Wallet
	err            error
	called         string
}

func (s *walletStorageStub) Deposit(context.Context, *model.User, *model.WalletOps) (*model.Wallet, error) {
	s.called = "deposit"
	return s.depositWallet, s.err
}
func (s *walletStorageStub) Withdraw(context.Context, *model.User, *model.WalletOps) (*model.Wallet, error) {
	s.called = "withdraw"
	return s.withdrawWallet, s.err
}
func (s *walletStorageStub) Balance(context.Context, *model.User) (*model.Wallet, error) {
	s.called = "balance"
	return s.balanceWallet, s.err
}
func (s *walletStorageStub) CreateWallet(context.Context, *model.User) (*model.Wallet, error) {
	s.called = "create"
	return s.createWallet, s.err
}
func (s *walletStorageStub) Exchange(context.Context, *model.User, *model.ExchangeOps, *model.ExchangeRate) (*model.Wallet, error) {
	s.called = "exchange"
	return s.exchangeWallet, s.err
}

type brokerStub struct{ called bool }

func (b *brokerStub) SendTransfer(context.Context, *model.Transfer) error {
	b.called = true
	return nil
}

func TestWalletServiceDelegatesToStorage(t *testing.T) {
	ctx := context.Background()
	user := &model.User{}
	wallet := &model.Wallet{}
	wantErr := errors.New("storage error")
	tests := []struct {
		name string
		call func(*WalletService) (*model.Wallet, error)
		want string
	}{
		{"create", func(s *WalletService) (*model.Wallet, error) { return s.CreateWallet(ctx, user) }, "create"},
		{"balance", func(s *WalletService) (*model.Wallet, error) { return s.Balance(ctx, user) }, "balance"},
		{"deposit", func(s *WalletService) (*model.Wallet, error) { return s.Deposit(ctx, user, &model.WalletOps{}) }, "deposit"},
		{"withdraw", func(s *WalletService) (*model.Wallet, error) { return s.Withdraw(ctx, user, &model.WalletOps{}) }, "withdraw"},
		{"exchange", func(s *WalletService) (*model.Wallet, error) {
			return s.Exchange(ctx, user, &model.ExchangeOps{}, &model.ExchangeRate{})
		}, "exchange"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &walletStorageStub{createWallet: wallet, balanceWallet: wallet, depositWallet: wallet, withdrawWallet: wallet, exchangeWallet: wallet, err: wantErr}
			got, err := tt.call(NewWalletService(storage, &brokerStub{}))
			if got != wallet || !errors.Is(err, wantErr) || storage.called != tt.want {
				t.Fatalf("got (%p, %v), storage call %q; want (%p, %v), call %q", got, err, storage.called, wallet, wantErr, tt.want)
			}
		})
	}
}

func TestWalletServiceRejectsNegativeAmounts(t *testing.T) {
	storage := &walletStorageStub{}
	broker := &brokerStub{}
	svc := NewWalletService(storage, broker)
	ctx := context.Background()
	if _, err := svc.Deposit(ctx, &model.User{}, &model.WalletOps{Amount: -1}); !errors.Is(err, model.ErrInvalidOperation) {
		t.Fatalf("Deposit error = %v, want %v", err, model.ErrInvalidOperation)
	}
	if _, err := svc.Withdraw(ctx, &model.User{}, &model.WalletOps{Amount: -1}); !errors.Is(err, model.ErrInvalidOperation) {
		t.Fatalf("Withdraw error = %v, want %v", err, model.ErrInvalidOperation)
	}
	if _, err := svc.Exchange(ctx, &model.User{}, &model.ExchangeOps{Amount: -1}, &model.ExchangeRate{}); !errors.Is(err, model.ErrInvalidOperation) {
		t.Fatalf("Exchange error = %v, want %v", err, model.ErrInvalidOperation)
	}
	if storage.called != "" || broker.called {
		t.Fatalf("invalid operation reached dependency: storage call %q, broker called %v", storage.called, broker.called)
	}
}

func TestWalletServiceRejectsMismatchedExchangeCurrencies(t *testing.T) {
	storage := &walletStorageStub{}
	op := &model.ExchangeOps{From: "USD", To: "GBP"}
	rate := &model.ExchangeRate{From: "EUR", To: "GBP"}
	svc := NewWalletService(storage, &brokerStub{})
	if _, err := svc.Exchange(context.Background(), &model.User{}, op, rate); !errors.Is(err, model.ErrInvalidOperation) {
		t.Fatalf("Exchange error = %v, want %v", err, model.ErrInvalidOperation)
	}
	if storage.called != "" {
		t.Fatalf("invalid exchange reached storage: call %q", storage.called)
	}
}
