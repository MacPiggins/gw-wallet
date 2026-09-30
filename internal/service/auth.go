package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/MacPiggins/gw-wallet/internal/model"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type AuthStorage interface {
	GetUser(ctx context.Context, u *model.User) (*model.User, error)
	AddUser(ctx context.Context, u *model.User) error
}

type AuthService struct {
	storage AuthStorage
	secret  []byte
}

func NewAuthService(storage AuthStorage, secret []byte) *AuthService {
	return &AuthService{storage: storage, secret: secret}
}

func (svc *AuthService) Register(ctx context.Context, u *model.User) error {
	_, err := svc.storage.GetUser(ctx, u)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			u.Password, err = hashPassword(ctx, u.Password)
			err := svc.storage.AddUser(ctx, u)
			if err != nil {
				slog.ErrorContext(ctx, "registration error", slog.Any("error", err))
			}
			return nil
		}
		slog.ErrorContext(ctx, "registration error", slog.Any("error", err))
		return err
	}
	return model.ErrUserAlreadyExists
}

func (svc *AuthService) Login(ctx context.Context, u *model.User) (string, error) {
	user, err := svc.storage.GetUser(ctx, u)
	if err != nil {
		slog.ErrorContext(ctx, "login error", slog.Any("error", err))
		return "", err
	}
	err = comparePassword(ctx, u.Password, user.Password)
	if err != nil {
		slog.ErrorContext(ctx, "login error", slog.Any("error", err))
		return "", err
	}
	claims := model.JwtClaims{
		UserName: u.UserName,
		Expires:  time.Now().Add(time.Minute * 5).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(svc.secret)
}

func (svc *AuthService) VerifyToken(token string) (*model.User, error) {
	claims := &model.JwtClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, model.ErrUnexpectedSigningMethod
		}
		return svc.secret, nil
	})
	if err != nil {
		return nil, model.ErrFailedToVerifyJwt
	}
	if !parsed.Valid {
		return nil, model.ErrFailedToVerifyJwt
	}
	if claims.Expires < time.Now().Unix() {
		return nil, model.ErrJwtExpired
	}

	return &model.User{UserName: claims.UserName}, nil
}

func hashPassword(ctx context.Context, password string) (string, error) {
	temp := sha256.Sum256([]byte(password))
	hash, err := bcrypt.GenerateFromPassword([]byte(base64.StdEncoding.EncodeToString(temp[:])), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "error while hashing password", slog.Any("error", err))
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

func comparePassword(ctx context.Context, password, hashPassword string) error {
	temp := sha256.Sum256([]byte(password))
	pass := []byte(base64.StdEncoding.EncodeToString(temp[:]))
	hashedPass, err := base64.StdEncoding.DecodeString(hashPassword)
	if err != nil {
		slog.ErrorContext(ctx, "error while comparing passwords", slog.Any("error", err))
		return err
	}
	err = bcrypt.CompareHashAndPassword(hashedPass, pass)
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		slog.ErrorContext(ctx, "wrong password error", slog.Any("error", err))
		return model.ErrWrongPassword
	}
	return err
}
