package model

import (
	"errors"
	"log/slog"
)

type User struct {
	UserName string 
	Password string
	Email    string
}

func (u *User) LogValue() slog.Value {
	return slog.StringValue(u.UserName)
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrWrongPassword     = errors.New("wrong password")
)
