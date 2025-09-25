package model

import "github.com/golang-jwt/jwt/v5"

type UserClaim struct {
	jwt.RegisteredClaims
	ID                string `json:"id"`
	Role              string `json:"role"`
	SID               string `json:"sid"`
	PasswordTemporary bool   `json:"passwordTemporary"`
}
