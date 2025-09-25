package model

import (
	"net/mail"
	"unicode"
)

type RegisterDTO struct {
	Validator
	Email string `json:"email"`
}

type LoginDTO struct {
	Validator
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshTokenDTO struct {
	Validator
	RefreshToken string `json:"refreshToken"`
}

type ChangePasswordDTO struct {
	Validator

	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}

type ResetPasswordDTO struct {
	Validator
	Email string `json:"email"`
}

func (dto *RegisterDTO) Validate() map[string]string {
	err := make(map[string]string)
	if dto.Email == "" {
		err["email"] = ErrEmptyField
	} else if _, pErr := mail.ParseAddress(dto.Email); pErr != nil {
		err["email"] = ErrInvalidField
	}

	return err
}

func (dto *LoginDTO) Validate() map[string]string {
	err := make(map[string]string)
	if dto.Email == "" {
		err["email"] = ErrEmptyField
	} else if _, pErr := mail.ParseAddress(dto.Email); pErr != nil {
		err["email"] = ErrInvalidField
	}

	if dto.Password == "" {
		err["password"] = ErrEmptyField
	}

	return err
}

func (dto *RefreshTokenDTO) Validate() map[string]string {
	err := make(map[string]string)
	if dto.RefreshToken == "" {
		err["refreshToken"] = ErrEmptyField
	}
	return err
}

func (dto *ResetPasswordDTO) Validate() map[string]string {
	err := make(map[string]string)
	if dto.Email == "" {
		err["email"] = ErrEmptyField
	}
	if _, pErr := mail.ParseAddress(dto.Email); pErr != nil {
		err["email"] = ErrInvalidField
	}
	return err
}

func (dto *ChangePasswordDTO) Validate() map[string]string {
	err := make(map[string]string)
	if dto.Password == "" {
		err["password"] = ErrEmptyField
	}
	if dto.Password == dto.NewPassword {
		err["newPassword"] = ErrInvalidField
	} else if dto.NewPassword == "" {
		err["newPassword"] = ErrEmptyField
	} else {
		hasLowercase := false
		hasUppercase := false
		hasDigit := false

		for _, char := range dto.NewPassword {
			if unicode.IsLower(char) {
				hasLowercase = true
			} else if unicode.IsUpper(char) {
				hasUppercase = true
			} else if unicode.IsDigit(char) {
				hasDigit = true
			}
		}

		if !hasLowercase || !hasUppercase || !hasDigit || len(dto.NewPassword) < 6 {
			err["newPassword"] = ErrInvalidField
		}
	}

	return err
}

type LoginResponse struct {
	TokenType    string `json:"tokenType"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type RefreshTokenResponse struct {
	TokenType    string `json:"tokenType"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type JwtDTO struct {
	ID                string `json:"id"`
	Role              string `json:"role"`
	SID               string `json:"sid"`
	PasswordTemporary bool   `json:"passwordTemporary"`
}
