package model

import (
	"errors"

	"github.com/golang-jwt/jwt"
)

type JwtClaims struct {
	jwt.StandardClaims
	UserName string
	Expires  int64
}

var (
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method error")
	ErrFailedToVerifyJwt       = errors.New("failed to verify jwt")
	ErrJwtExpired              = errors.New("auth token expired")
)
