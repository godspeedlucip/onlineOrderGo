package errors

import (
	"errors"
	"net/http"

	"go-baseline-skeleton/internal/baseline/domain"
)

type HTTPErrorMapper struct{}

func NewHTTPErrorMapper() *HTTPErrorMapper {
	return &HTTPErrorMapper{}
}

func (m *HTTPErrorMapper) ToHTTP(err error) (status int, code string, message string) {
	if err == nil {
		return http.StatusOK, string(domain.CodeOK), "success"
	}

	var bizErr *domain.BizError
	if errors.As(err, &bizErr) {
		switch bizErr.Code {
		case domain.CodeInvalidArgument:
			return http.StatusBadRequest, string(bizErr.Code), bizErr.Message
		case domain.CodeUnauthorized:
			return http.StatusUnauthorized, string(bizErr.Code), bizErr.Message
		case domain.CodeForbidden:
			return http.StatusForbidden, string(bizErr.Code), bizErr.Message
		case domain.CodeConflict:
			return http.StatusConflict, string(bizErr.Code), bizErr.Message
		case domain.CodeServiceUnavailable:
			return http.StatusServiceUnavailable, string(bizErr.Code), bizErr.Message
		default:
			return http.StatusInternalServerError, string(bizErr.Code), bizErr.Message
		}
	}

	return http.StatusInternalServerError, string(domain.CodeInternal), "internal error"
}