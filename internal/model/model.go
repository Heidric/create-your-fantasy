package model

type Validator interface {
	Validate() map[string]string
}

const (
	ErrEmptyField   = "EMPTY"
	ErrInvalidField = "INVALID_VALUE"
)
