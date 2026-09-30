package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MacPiggins/gw-wallet/internal/model"
	"github.com/golang-jwt/jwt"
)

type authStorageMock struct {
	user   *model.User
	getErr error
	adds   int
}

func (m *authStorageMock) GetUser(_ context.Context, _ *model.User) (*model.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.user == nil {
		return nil, model.ErrUserNotFound
	}
	return m.user, nil
}

func (m *authStorageMock) AddUser(_ context.Context, _ *model.User) error {
	m.adds++
	return nil
}

func TestRegister(t *testing.T) {
	t.Run("registers new user with hashed password", func(t *testing.T) {
		storage := &authStorageMock{}
		user := &model.User{UserName: "alice", Password: "password"}
		if err := NewAuthService(storage, nil).Register(context.Background(), user); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if storage.adds != 1 {
			t.Fatalf("AddUser calls = %d, want 1", storage.adds)
		}
		if err := comparePassword(context.Background(), "password", user.Password); err != nil {
			t.Fatalf("registered password did not verify: %v", err)
		}
	})

	t.Run("returns already exists", func(t *testing.T) {
		storage := &authStorageMock{user: &model.User{UserName: "alice"}}
		if err := NewAuthService(storage, nil).Register(context.Background(), &model.User{}); !errors.Is(err, model.ErrUserAlreadyExists) {
			t.Fatalf("Register() error = %v, want ErrUserAlreadyExists", err)
		}
		if storage.adds != 0 {
			t.Fatalf("AddUser calls = %d, want 0", storage.adds)
		}
	})

	t.Run("returns lookup error", func(t *testing.T) {
		want := errors.New("storage failure")
		if err := NewAuthService(&authStorageMock{getErr: want}, nil).Register(context.Background(), &model.User{}); !errors.Is(err, want) {
			t.Fatalf("Register() error = %v, want %v", err, want)
		}
	})
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	password, err := hashPassword(ctx, "secret")
	if err != nil {
		t.Fatal(err)
	}
	service := NewAuthService(&authStorageMock{user: &model.User{UserName: "alice", Password: password}}, []byte("key"))

	t.Run("issues verifiable token", func(t *testing.T) {
		token, err := service.Login(ctx, &model.User{UserName: "alice", Password: "secret"})
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		user, err := service.VerifyToken(token)
		if err != nil || user.UserName != "alice" {
			t.Fatalf("VerifyToken() = (%v, %v), want alice", user, err)
		}
	})

	t.Run("rejects wrong password", func(t *testing.T) {
		_, err := service.Login(ctx, &model.User{UserName: "alice", Password: "wrong"})
		if !errors.Is(err, model.ErrWrongPassword) {
			t.Fatalf("Login() error = %v, want ErrWrongPassword", err)
		}
	})

	t.Run("returns lookup error", func(t *testing.T) {
		want := errors.New("storage failure")
		_, err := NewAuthService(&authStorageMock{getErr: want}, nil).Login(ctx, &model.User{})
		if !errors.Is(err, want) {
			t.Fatalf("Login() error = %v, want %v", err, want)
		}
	})
}

func TestVerifyToken(t *testing.T) {
	secret := []byte("key")
	service := NewAuthService(&authStorageMock{}, secret)
	validToken := func(exp time.Time) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, model.JwtClaims{UserName: "alice", Expires: exp.Unix()})
		encoded, err := token.SignedString(secret)
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}

	t.Run("accepts valid token", func(t *testing.T) {
		user, err := service.VerifyToken(validToken(time.Now().Add(time.Minute)))
		if err != nil || user.UserName != "alice" {
			t.Fatalf("VerifyToken() = (%v, %v), want alice", user, err)
		}
	})
	t.Run("rejects malformed token", func(t *testing.T) {
		if _, err := service.VerifyToken("invalid"); !errors.Is(err, model.ErrFailedToVerifyJwt) {
			t.Fatalf("VerifyToken() error = %v, want ErrFailedToVerifyJwt", err)
		}
	})
	t.Run("rejects expired token", func(t *testing.T) {
		if _, err := service.VerifyToken(validToken(time.Now().Add(-time.Minute))); !errors.Is(err, model.ErrJwtExpired) {
			t.Fatalf("VerifyToken() error = %v, want ErrJwtExpired", err)
		}
	})
	t.Run("rejects token signed with another secret", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, model.JwtClaims{UserName: "alice", Expires: time.Now().Add(time.Minute).Unix()})
		encoded, err := token.SignedString([]byte("other-key"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.VerifyToken(encoded); !errors.Is(err, model.ErrFailedToVerifyJwt) {
			t.Fatalf("VerifyToken() error = %v, want ErrFailedToVerifyJwt", err)
		}
	})
}

func TestComparePassword(t *testing.T) {
	hash, err := hashPassword(context.Background(), "correct")
	if err != nil {
		t.Fatal(err)
	}
	if err := comparePassword(context.Background(), "correct", hash); err != nil {
		t.Fatalf("comparePassword() error = %v", err)
	}
	if err := comparePassword(context.Background(), "incorrect", hash); !errors.Is(err, model.ErrWrongPassword) {
		t.Fatalf("comparePassword() error = %v, want ErrWrongPassword", err)
	}
	if err := comparePassword(context.Background(), "correct", "not-base64!"); err == nil {
		t.Fatal("comparePassword() accepted malformed hash")
	}
}
