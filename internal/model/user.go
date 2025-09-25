package model

import (
	"net/mail"
	"strings"
	"time"
)

var Admin = "Admin"
var Moderator = "Moderator"
var Gm = "Gm"
var Player = "Player"

var PossibleRoles = []string{Admin, Moderator, Gm, Player}

type User struct {
	ID                string `db:"id" json:"id"`
	Email             string `db:"email" json:"email"`
	Name              string `db:"name" json:"name,omitempty"`
	Role              string `db:"role" json:"role"`
	Avatar            string `db:"avatar" json:"avatar,omitempty"`
	Password          string `db:"password_hash" json:"-"`
	PasswordTemporary bool   `db:"password_temporary" json:"passwordTemporary"`
}

type Session struct {
	ID                string    `db:"id" json:"id"`
	UserID            string    `db:"user_id" json:"userId"`
	AccessToken       string    `db:"access_token" json:"accessToken"`
	RefreshToken      string    `db:"refresh_token" json:"refreshToken"`
	PasswordTemporary bool      `db:"password_temporary" json:"passwordTemporary"`
	ExpiresAt         time.Time `db:"expires_at" json:"expiresAt"`
}

type RegisterUserDTO struct {
	Validator

	Email string `json:"email"`
}

func (dto *RegisterUserDTO) Validate() map[string]string {
	err := make(map[string]string)

	if strings.TrimSpace(dto.Email) == "" {
		err["email"] = ErrEmptyField
	} else if _, pErr := mail.ParseAddress(dto.Email); pErr != nil {
		err["email"] = ErrInvalidField
	}

	return err
}
