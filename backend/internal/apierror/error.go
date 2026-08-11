package apierror

import (
	"errors"
	"net/http"
)

const (
	CodeAuthRequired        = 1001
	CodeAccountDisabled     = 1002
	CodeForbidden           = 1003
	CodeEntitlementMissing  = 2001
	CodeInsufficientCredit  = 2002
	CodeRedemptionInvalid   = 3001
	CodeRedemptionUsed      = 3002
	CodeRedemptionExpired   = 3003
	CodeProductMismatch     = 3004
	CodeRateLimited         = 4001
	CodeIdempotencyConflict = 4002
	CodeGenerationFailed    = 5001
	CodeInvalidInput        = 6001
	CodeAssetNotFound       = 6002
	CodePromptNotFound      = 6003
	CodeTaskNotFound        = 6004
	CodeRetouchIneligible   = 7001
	CodeRetouchExists       = 7002
	CodeRetouchNotFound     = 7003
	CodeRetouchState        = 7004
	CodeRetouchQuote        = 7005
	CodeRetouchRevision     = 7006
	CodeAIUnavailable       = 8001
	CodeAICapability        = 8002
	CodeAIProvider          = 8003
	CodeInternal            = 9999
)

type AppError struct {
	HTTPStatus int
	Code       int
	Message    string
	Details    any
	Cause      error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(status, code int, message string, details any) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: message, Details: details}
}

func Wrap(err error, status, code int, message string) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: message, Cause: err}
}

func Invalid(message string, details any) *AppError {
	if message == "" {
		message = "请求参数无效"
	}
	return New(http.StatusBadRequest, CodeInvalidInput, message, details)
}

func AuthRequired() *AppError {
	return New(http.StatusUnauthorized, CodeAuthRequired, "登录状态已失效，请重新登录", nil)
}

func InvalidCredentials() *AppError {
	return New(http.StatusUnauthorized, CodeAuthRequired, "邮箱或密码错误", nil)
}

func Disabled() *AppError {
	return New(http.StatusForbidden, CodeAccountDisabled, "账号已停用", nil)
}

func Forbidden() *AppError {
	return New(http.StatusForbidden, CodeForbidden, "没有权限执行此操作", nil)
}

func Internal(err error) *AppError {
	return Wrap(err, http.StatusInternalServerError, CodeInternal, "服务暂时不可用")
}

func As(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
