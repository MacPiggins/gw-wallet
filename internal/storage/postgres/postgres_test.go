package postgres

import (
	"context"
	"testing"

	"github.com/MacPiggins/gw-wallet/internal/model"

	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func newTestDB(t *testing.T, initScript bool) (*PostgresDB, func()) {
	t.Helper()
	ctx := context.Background()

	options := []testcontainers.ContainerCustomizer{
		postgrescontainer.WithDatabase("wallet"),
		postgrescontainer.WithUsername("postgres"),
		postgrescontainer.WithPassword("postgres"),
		postgrescontainer.BasicWaitStrategies(),
	}
	if initScript {
		options = append(options, postgrescontainer.WithInitScripts("testdata/test.sql"))
	}

	container, err := postgrescontainer.Run(ctx, "postgres:16-alpine", options...)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	connstr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("get postgres connection string: %v", err)
	}

	db, err := New(ctx, connstr)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("create database: %v", err)
	}

	cleanup := func() {
		db.Close()
		container.Terminate(ctx)
	}
	return db, cleanup
}

func TestAutoMigrate(t *testing.T) {
	db, cleanup := newTestDB(t, false)
	defer cleanup()

	if err := db.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
}

func TestWalletOperations(t *testing.T) {
	db, cleanup := newTestDB(t, true)
	defer cleanup()
	ctx := context.Background()
	user := &model.User{
		UserName: "alice",
		Password: "secret",
		Email:    "alice@example.com",
	}
	if err := db.AddUser(ctx, user); err != nil {
		t.Fatalf("add user: %v", err)
	}

	wallet, err := db.CreateWallet(ctx, user)
	if err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	if wallet.UserName != user.UserName {
		t.Fatalf("wallet username = %q, want %q", wallet.UserName, user.UserName)
	}

	wallet, err = db.Deposit(ctx, user, &model.WalletOps{Currency: "usd", Amount: 25})
	if err != nil {
		t.Fatalf("deposit: %v", err)
	}
	if wallet.Balance.Usd != 25 {
		t.Fatalf("USD balance = %v, want 25", wallet.Balance.Usd)
	}

	wallet, err = db.Withdraw(ctx, user, &model.WalletOps{Currency: "usd", Amount: 5})
	if err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if wallet.Balance.Usd != 20 {
		t.Fatalf("USD balance = %v, want 20", wallet.Balance.Usd)
	}

	wallet, err = db.Balance(ctx, user)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if wallet.Balance.Usd != 20 {
		t.Fatalf("USD balance = %v, want 20", wallet.Balance.Usd)
	}

	wallet, err = db.Exchange(ctx, user,
		&model.ExchangeOps{From: "usd", To: "eur", Amount: 10},
		&model.ExchangeRate{Rate: 0.918367},
	)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if wallet.Balance.Usd != 10 || wallet.Balance.Eur != 9 {
		t.Fatalf("balances after exchange = USD %v, EUR %v; want USD 10, EUR 9", wallet.Balance.Usd, wallet.Balance.Eur)
	}

	wallet, err = db.Exchange(ctx, user,
		&model.ExchangeOps{From: "eur", To: "usd", Amount: 9},
		&model.ExchangeRate{Rate: 1.088889},
	)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if wallet.Balance.Usd != 20 || wallet.Balance.Eur != 0 {
		t.Fatalf("balances after exchange = USD %v, EUR %v; want USD 20, EUR 0", wallet.Balance.Usd, wallet.Balance.Eur)
	}
}

func TestUserOperations(t *testing.T) {
	db, cleanup := newTestDB(t, true)
	defer cleanup()
	ctx := context.Background()
	want := &model.User{UserName: "alice", Password: "secret", Email: "alice@example.com"}

	if err := db.AddUser(ctx, want); err != nil {
		t.Fatalf("add user: %v", err)
	}
	got, err := db.GetUser(ctx, &model.User{UserName: want.UserName})
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.UserName != want.UserName || got.Password != want.Password || got.Email != want.Email {
		t.Fatalf("got user %+v, want %+v", got, want)
	}
}
