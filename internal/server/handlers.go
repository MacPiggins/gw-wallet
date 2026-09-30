package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MacPiggins/gw-wallet/internal/model"
	"github.com/MacPiggins/gw-wallet/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Broker interface {
	SendTransfer(ctx context.Context, message *model.Transfer) error
	SendEvent(ctx context.Context, message *model.Event) error
}

type Handler struct {
	broker      Broker
	AuthSvc     *service.AuthService
	ExchangeSvc *service.ExchangeService
	WalletSvc   *service.WalletService
}

func NewHandler(auth *service.AuthService, exchange *service.ExchangeService, wallet *service.WalletService, broker Broker) *Handler {
	return &Handler{AuthSvc: auth, ExchangeSvc: exchange, WalletSvc: wallet, broker: broker}
}

// Register godoc
// @Summary Register a user
// @Description Creates a user account and wallet.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object true "Registration details"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /register [post]
func (handler *Handler) Register(ctx *gin.Context) {
	newEvent := handler.newEvent("user_registered")
	defer func() { handler.sendEvent(ctx, newEvent) }()

	type Request struct {
		UserName string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
	}
	var data Request
	if err := ctx.ShouldBind(&data); err != nil {
		slog.ErrorContext(ctx, "got invalid request", slog.Any("error", err))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "inappropriate json data"})
		newEvent.Status = "error"
		return
	}
	user := &model.User{UserName: data.UserName, Password: data.Password, Email: data.Email}
	err := handler.AuthSvc.Register(ctx, user)
	if err != nil {
		if errors.Is(err, model.ErrUserAlreadyExists) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Username or email already exists"})
			newEvent.Status = "error"
			return
		}
		slog.ErrorContext(ctx, "registration error", slog.Any("error", err))
		return
	}
	_, err = handler.WalletSvc.CreateWallet(ctx, user)
	if err != nil {
		slog.ErrorContext(ctx, "create wallet error", slog.Any("error", err))
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
	newEvent.Status = "ok"
}

// Login godoc
// @Summary Log in
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object true "Login credentials"
// @Success 201 {object} map[string]string
// @Failure 400,401 {object} map[string]string
// @Router /login [post]
func (handler *Handler) Login(ctx *gin.Context) {
	newEvent := handler.newEvent("user_login")
	defer func() { handler.sendEvent(ctx, newEvent) }()
	type Request struct {
		UserName string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var data Request
	if err := ctx.ShouldBind(&data); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "inappropriate json data"})
		newEvent.Status = "error"
		return
	}
	user := &model.User{UserName: data.UserName, Password: data.Password}
	token, err := handler.AuthSvc.Login(ctx, user)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		newEvent.Status = "error"
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"token": token})
	newEvent.Status = "ok"
}

// CheckAuth godoc
// @Summary Authenticate a request
// @Tags auth
// @Security BearerAuth
// @Success 200
// @Failure 401 {object} map[string]string
// @Router /auth [get]
func (handler *Handler) CheckAuth(ctx *gin.Context) {
	newEvent := handler.newEvent("auth_checked")
	defer func() { handler.sendEvent(ctx, newEvent) }()
	authorization := ctx.GetHeader("Authorization")
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		ctx.Abort()
		newEvent.Status = "error"
		return
	}

	user, err := handler.AuthSvc.VerifyToken(parts[1])
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		ctx.Abort()
		newEvent.Status = "error"
		return
	}

	ctx.Set("user", user)
	newEvent.Status = "ok"
	ctx.Next()
}

// Balance godoc
// @Summary Get wallet balance
// @Tags wallet
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /wallet/balance [get]
func (handler *Handler) Balance(ctx *gin.Context) {
	newEvent := handler.newEvent("wallet_balance")
	defer func() { handler.sendEvent(ctx, newEvent) }()
	u, ok := ctx.Get("user")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		newEvent.Status = "error"
		return
	}
	user, ok := u.(*model.User)
	wallet, err := handler.WalletSvc.Balance(ctx, user)
	if err != nil {
		// error
	}
	ctx.JSON(http.StatusOK, gin.H{"balance": wallet.Balance})
	newEvent.Status = "ok"
}

// Deposit godoc
// @Summary Deposit funds
// @Tags wallet
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param operation body model.WalletOps true "Deposit details"
// @Success 200 {object} map[string]interface{}
// @Failure 400,401 {object} map[string]string
// @Router /wallet/deposit [post]
func (handler *Handler) Deposit(ctx *gin.Context) {
	newEvent := handler.newEvent("wallet_deposit")
	defer func() { handler.sendEvent(ctx, newEvent) }()
	u, ok := ctx.Get("user")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		newEvent.Status = "error"
		return
	}
	user, ok := u.(*model.User)

	var op model.WalletOps
	if err := ctx.ShouldBind(&op); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "inappropriate json data"})
		newEvent.Status = "error"
		return
	}

	wallet, err := handler.WalletSvc.Deposit(ctx, user, &op)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds or invalid amount"})
		newEvent.Status = "error"
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Deposit successful", "new_balance": wallet.Balance})
	newEvent.Status = "ok"
}

// Withdraw godoc
// @Summary Withdraw funds
// @Tags wallet
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param operation body model.WalletOps true "Withdrawal details"
// @Success 200 {object} map[string]interface{}
// @Failure 400,401 {object} map[string]string
// @Router /wallet/withdraw [post]
func (handler *Handler) Withdraw(ctx *gin.Context) {
	newEvent := handler.newEvent("wallet_withdraw")
	defer func() { handler.sendEvent(ctx, newEvent) }()
	u, ok := ctx.Get("user")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		newEvent.Status = "error"
		return
	}
	user, ok := u.(*model.User)

	var op model.WalletOps
	if err := ctx.ShouldBind(&op); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "inappropriate json data"})
		newEvent.Status = "error"
		return
	}

	wallet, err := handler.WalletSvc.Withdraw(ctx, user, &op)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds or invalid amount"})
		newEvent.Status = "error"
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Withdrawal successful", "new_balance": wallet.Balance})
	newEvent.Status = "ok"
}

// Rates godoc
// @Summary Get exchange rates
// @Tags exchange
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /exchange/rates [get]
func (handler *Handler) Rates(ctx *gin.Context) {
	newEvent := handler.newEvent("exchange_rates")
	defer func() { handler.sendEvent(ctx, newEvent) }()
	rates, err := handler.ExchangeSvc.Rates(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to retrieve exchange rates", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rates"})
		newEvent.Status = "error"
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"rates": rates})
	newEvent.Status = "ok"
}

// Exchange godoc
// @Summary Exchange currencies
// @Tags exchange
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param operation body model.ExchangeOps true "Exchange details"
// @Success 200 {object} map[string]interface{}
// @Failure 400,401,500 {object} map[string]string
// @Router /exchange [post]
func (handler *Handler) Exchange(ctx *gin.Context) {
	newEvent := handler.newEvent("wallet_exchange")
	defer func() { handler.sendEvent(ctx, newEvent) }()
	u, ok := ctx.Get("user")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		newEvent.Status = "error"
		return
	}
	user, ok := u.(*model.User)

	op := &model.ExchangeOps{}
	if err := ctx.ShouldBind(op); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "inappropriate json data"})
		newEvent.Status = "error"
		return
	}

	// TODO check cache before making a request
	rate, err := handler.ExchangeSvc.Rate(ctx, op.From, op.To)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		newEvent.Status = "error"
		return
	}

	wallet, err := handler.WalletSvc.Exchange(ctx, user, op, rate)

	if err != nil {
		// error
		newEvent.Status = "error"
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Exchange successful", "exchanged_amount": rate.Rate * float32(op.Amount), "new_balance": wallet.Balance})
	newEvent.Status = "ok"
}

func (handler *Handler) newEvent(eventType string) *model.Event {
	return &model.Event{ID: uuid.NewString(), Type: eventType, OccurredAt: time.Now()}
}

func (handler *Handler) sendEvent(ctx *gin.Context, event *model.Event) {
	if handler.broker == nil {
		slog.ErrorContext(ctx, "failed to send event, broker is nil")
		return
	}
	err := handler.broker.SendEvent(ctx, event)
	if err != nil {
		slog.ErrorContext(ctx, "error sending event to broker", slog.Any("error", err))
	}
}
