package service

import (
	"context"
	"time"

	"github.com/MacPiggins/gw-wallet/internal/model"
	"github.com/google/uuid"
)

//TODO
//logging

type WalletStorage interface {
	Deposit(ctx context.Context, u *model.User, op *model.WalletOps) (*model.Wallet, error)
	Withdraw(ctx context.Context, u *model.User, op *model.WalletOps) (*model.Wallet, error)
	Balance(ctx context.Context, u *model.User) (*model.Wallet, error)
	CreateWallet(ctx context.Context, u *model.User) (*model.Wallet, error)
	Exchange(ctx context.Context, u *model.User, op *model.ExchangeOps, rate *model.ExchangeRate) (*model.Wallet, error)
}

type Broker interface {
	SendTransfer(ctx context.Context, message *model.Transfer) error
}

type WalletService struct {
	storage WalletStorage
	broker  Broker
}

func NewWalletService(storage WalletStorage, broker Broker) *WalletService {
	return &WalletService{storage: storage, broker: broker}
}

func (svc *WalletService) CreateWallet(ctx context.Context, u *model.User) (*model.Wallet, error) {
	return svc.storage.CreateWallet(ctx, u)
}

func (svc *WalletService) Balance(ctx context.Context, u *model.User) (*model.Wallet, error) {
	return svc.storage.Balance(ctx, u)
}

func (svc *WalletService) Deposit(ctx context.Context, u *model.User, op *model.WalletOps) (*model.Wallet, error) {
	if op.Amount < 0 {
		return nil, model.ErrInvalidOperation
	}
	if op.Amount > 30000 {
		svc.broker.SendTransfer(ctx, &model.Transfer{
			TransactionID: uuid.New().String(),
			Sender:        u.UserName,
			Receiver:      u.UserName,
			Amount:        op.Amount,
			Currency:      op.Currency,
			OccurredAt:    time.Now(),
			ReceivedAt:    time.Unix(0, 0),
		})
	}
	return svc.storage.Deposit(ctx, u, op)
}

func (svc *WalletService) Withdraw(ctx context.Context, u *model.User, op *model.WalletOps) (*model.Wallet, error) {
	if op.Amount < 0 {
		return nil, model.ErrInvalidOperation
	}
	if op.Amount > 30000 {
		svc.broker.SendTransfer(ctx, &model.Transfer{
			TransactionID: uuid.New().String(),
			Sender:        u.UserName,
			Receiver:      u.UserName,
			Amount:        op.Amount,
			Currency:      op.Currency,
			OccurredAt:    time.Now(),
			ReceivedAt:    time.Unix(0, 0),
		})
	}
	return svc.storage.Withdraw(ctx, u, op)
}

func (svc *WalletService) Exchange(ctx context.Context, u *model.User, op *model.ExchangeOps, rate *model.ExchangeRate) (*model.Wallet, error) {
	if op.Amount < 0 || op.From != rate.From || op.To != rate.To {
		return nil, model.ErrInvalidOperation
	}
	return svc.storage.Exchange(ctx, u, op, rate)
}
