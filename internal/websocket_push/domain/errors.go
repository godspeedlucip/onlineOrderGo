package domain

import "fmt"

type ErrorCode string

const (
	CodeInvalidArgument ErrorCode = "COMMON_400"
	CodeConflict        ErrorCode = "COMMON_409"
	CodeUnauthorized    ErrorCode = "AUTH_401"
	CodeInternal        ErrorCode = "COMMON_500"
)

type BizError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *BizError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s:%s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s:%s cause=%v", e.Code, e.Message, e.Cause)
}

func (e *BizError) Unwrap() error { return e.Cause }

func NewBizError(code ErrorCode, msg string, cause error) *BizError {
	return &BizError{Code: code, Message: msg, Cause: cause}
}
