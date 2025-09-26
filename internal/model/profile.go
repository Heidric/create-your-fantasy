package model

import (
	"encoding/base64"
	"net/http"
)

type ProfileResponse struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Avatar string `json:"avatar"`
}

type UpdateProfileDTO struct {
	Validator
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Avatar      string `json:"avatar"`
}

func (dto UpdateProfileDTO) Validate() map[string]string {
	errs := map[string]string{}

	if dto.Avatar == "" {
		return errs
	}

	if dto.ContentType == "" {
		errs["contentType"] = ErrEmptyField
		return errs
	}

	if len(dto.ContentType) < 6 || dto.ContentType[:6] != "image/" {
		errs["contentType"] = ErrInvalidField
		return errs
	}

	raw, decErr := base64.StdEncoding.DecodeString(dto.Avatar)
	if decErr != nil {
		errs["avatar"] = ErrInvalidField
		return errs
	}
	if len(raw) > 3*1024*1024 {
		errs["avatar"] = ErrInvalidField
		return errs
	}

	detected := http.DetectContentType(raw)
	if detected[:6] != "image/" || detected != dto.ContentType {
		errs["avatar"] = ErrInvalidField
		return errs
	}

	return errs
}
