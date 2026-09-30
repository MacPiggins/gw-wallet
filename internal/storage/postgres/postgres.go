package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/MacPiggins/gw-wallet/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type PostgresDB struct {
	pool    *pgxpool.Pool
	connstr string
	close   chan struct{}
}

func New(ctx context.Context, connstr string) (*PostgresDB, error) {
	pool, err := pgxpool.New(ctx, connstr)
	if err != nil {
		slog.ErrorContext(ctx, "unable to connect to postgres error", slog.Any("error", err))
		return nil, err
	}

	close := make(chan struct{}, 1)
	db := &PostgresDB{pool: pool, connstr: connstr, close: close}
	go db.maintainConnection()
	return db, nil
}

func (db *PostgresDB) AutoMigrate(ctx context.Context) error {
	migrationDB := stdlib.OpenDBFromPool(db.pool)
	defer migrationDB.Close()
	migrations, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		db.pool.Close()
		return fmt.Errorf("load postgres migrations: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, migrationDB, migrations)
	if err != nil {
		db.pool.Close()
		return fmt.Errorf("configure postgres migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		db.pool.Close()
		return fmt.Errorf("run postgres migrations: %w", err)
	}
	return nil
}

func (db *PostgresDB) maintainConnection() {
	for {
		select {
		case <-db.close:
			db.pool.Close()
			return
		case <-time.After(5 * time.Second):
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			err := db.pool.Ping(ctx)
			if err != nil {
				slog.Error("postgres connection error, trying to reconnect", slog.Any("error", err))
				db.pool.Close()
				db.pool, err = pgxpool.New(ctx, db.connstr)
				if err != nil {
					slog.Error("postgres failed to reconnect, trying again in 5 sec", slog.Any("error", err))
				}
				slog.Info("postgres successfully reconnected")
			}
			cancel()
		}
	}
}

func (db *PostgresDB) Close() {
	close(db.close)
}

// wallet interface
func (db *PostgresDB) Deposit(ctx context.Context, u *model.User, op *model.WalletOps) (*model.Wallet, error) {
	newWallet := &model.Wallet{}
	sql := fmt.Sprintf("UPDATE Wallets set %s = %s + $1 where username = $2 returning username, usd, rub, eur;", op.Currency, op.Currency)
	err := db.pool.QueryRow(ctx, sql, op.Amount, u.UserName).Scan(&newWallet.UserName, &newWallet.Balance.Usd, &newWallet.Balance.Rub, &newWallet.Balance.Eur)
	return newWallet, err
}
func (db *PostgresDB) Withdraw(ctx context.Context, u *model.User, op *model.WalletOps) (*model.Wallet, error) {
	newWallet := &model.Wallet{}
	sql := fmt.Sprintf("UPDATE Wallets set %s = %s - $1 where username = $2 returning username, usd, rub, eur;", op.Currency, op.Currency)
	err := db.pool.QueryRow(ctx, sql, op.Amount, u.UserName).Scan(&newWallet.UserName, &newWallet.Balance.Usd, &newWallet.Balance.Rub, &newWallet.Balance.Eur)
	return newWallet, err
}
func (db *PostgresDB) Balance(ctx context.Context, u *model.User) (*model.Wallet, error) {
	newWallet := &model.Wallet{}
	err := db.pool.QueryRow(ctx, "select username, usd, rub, eur from Wallets where username = $1;", u.UserName).Scan(&newWallet.UserName, &newWallet.Balance.Usd, &newWallet.Balance.Rub, &newWallet.Balance.Eur)
	return newWallet, err
}
func (db *PostgresDB) CreateWallet(ctx context.Context, u *model.User) (*model.Wallet, error) {
	newWallet := &model.Wallet{}
	err := db.pool.QueryRow(ctx, "INSERT INTO wallets (Username) VALUES ($1) returning username, usd, rub, eur;", u.UserName).Scan(&newWallet.UserName, &newWallet.Balance.Usd, &newWallet.Balance.Rub, &newWallet.Balance.Eur)
	return newWallet, err
}
func (db *PostgresDB) Exchange(ctx context.Context, u *model.User, op *model.ExchangeOps, rate *model.ExchangeRate) (*model.Wallet, error) {
	newWallet := &model.Wallet{}
	sql := fmt.Sprintf("UPDATE Wallets set %s = %s - $1, %s = %s + $1*$2::numeric WHERE Username = $3 returning username, usd, rub, eur;", op.From, op.From, op.To, op.To)
	err := db.pool.QueryRow(ctx, sql, op.Amount, rate.Rate, u.UserName).Scan(&newWallet.UserName, &newWallet.Balance.Usd, &newWallet.Balance.Rub, &newWallet.Balance.Eur)
	return newWallet, err
}

// auth interface
func (db *PostgresDB) GetUser(ctx context.Context, u *model.User) (*model.User, error) {
	newUser := &model.User{}
	err := db.pool.QueryRow(ctx, "SELECT username, password, Email from Users WHERE Username = $1;", u.UserName).Scan(&newUser.UserName, &newUser.Password, &newUser.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrUserNotFound
	}
	return newUser, err
}
func (db *PostgresDB) AddUser(ctx context.Context, u *model.User) error {
	_, err := db.pool.Exec(ctx, "INSERT INTO Users(Username, Password, Email) VALUES ($1, $2, $3);", u.UserName, u.Password, u.Email)
	return err
}
