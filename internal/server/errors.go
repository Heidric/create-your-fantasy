package server

import (
	"encoding/json"
	"net/http"
)

const (
	ErrTokenInvalid        = "TOKEN_INVALID"
	ErrUserNotFound        = "USER_NOT_FOUND"
	ErrEmailNotFound       = "EMAIL_NOT_FOUND"
	ErrEmailNotUnique      = "EMAIL_NOT_UNIQUE"
	ErrUserCannotBeDeleted = "USER_CANNOT_BE_DELETED"
	ErrKeyNotUnique        = "KEY_NOT_UNIQUE"
)

type CommonError struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

type Validation struct {
	Title  string            `json:"title"`
	Status int               `json:"status"`
	Detail string            `json:"detail"`
	Code   string            `json:"code"`
	Errors map[string]string `json:"errors"`
}

func ParsingError(w http.ResponseWriter) {
	res := CommonError{
		Title:  "Parsing error occurred",
		Status: http.StatusBadRequest,
		Detail: "Parsing error",
		Code:   "PARSING_ERROR",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(res)
}

func ValidationError(w http.ResponseWriter, err map[string]string) {
	res := Validation{
		Title:  "One or more model validation errors occurred",
		Status: http.StatusUnprocessableEntity,
		Detail: "See the errors property for details",
		Code:   "VALIDATION_ERROR",
		Errors: err,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(res)
}

func LogicError(w http.ResponseWriter, code string) {
	res := CommonError{
		Title:  "Logic error occurred",
		Status: http.StatusBadRequest,
		Detail: "Logic error",
		Code:   code,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(res)
}

func ConflictError(w http.ResponseWriter) {
	res := CommonError{
		Title:  "Conflict error",
		Status: http.StatusConflict,
		Detail: "Conflict error",
		Code:   "CONFLICT",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(res)
}

func UnauthorizedError(w http.ResponseWriter) {
	res := CommonError{
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: "Unauthorized",
		Code:   "UNAUTHORIZED",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(res)
}

func ForbiddenError(w http.ResponseWriter) {
	res := CommonError{
		Title:  "Forbidden",
		Status: http.StatusForbidden,
		Detail: "Forbidden",
		Code:   "FORBIDDEN",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(res)
}

func NotFoundError(w http.ResponseWriter) {
	res := CommonError{
		Title:  "Endpoint not found",
		Status: http.StatusNotFound,
		Detail: "Not found",
		Code:   "ENDPOINT_NOT_FOUND",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(res)
}

func InternalError(w http.ResponseWriter) {
	res := CommonError{
		Title:  "Resource temporarily unavailable",
		Status: http.StatusInternalServerError,
		Detail: "Resource temporarily unavailable",
		Code:   "UNKNOWN_ERROR",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(res)
}

func BadRequestError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(CommonError{
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: "Bad request",
		Code:   "BAD_REQUEST",
	})
}
