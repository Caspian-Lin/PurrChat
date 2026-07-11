package onebot

import "errors"

type RetCode int

const (
	RetCodeOK RetCode = 0

	RetCodeBadRequest         RetCode = 10001
	RetCodeInvalidParams      RetCode = 10002
	RetCodeUnknownAction      RetCode = 10003
	RetCodeUnsupportedSegment RetCode = 10004

	RetCodeUnauthenticated      RetCode = 20001
	RetCodeCredentialRevoked    RetCode = 20002
	RetCodeCredentialExpired    RetCode = 20003
	RetCodePermissionDenied     RetCode = 30001
	RetCodeCapabilityRequired   RetCode = 30002
	RetCodeInstallationInactive RetCode = 30003

	RetCodeResourceNotFound RetCode = 40001
	RetCodeConflict         RetCode = 40002

	RetCodeRateLimited RetCode = 50001
	RetCodeInternal    RetCode = 90001
)

type Error struct {
	Code    RetCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func NewError(code RetCode, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func AsError(err error) *Error {
	var protocolErr *Error
	if errors.As(err, &protocolErr) {
		return protocolErr
	}
	return NewError(RetCodeInternal, "internal error", err)
}

func (code RetCode) Category() string {
	switch {
	case code == RetCodeOK:
		return "success"
	case code >= 10000 && code < 20000:
		return "request"
	case code >= 20000 && code < 30000:
		return "authentication"
	case code >= 30000 && code < 40000:
		return "permission"
	case code >= 40000 && code < 50000:
		return "resource"
	case code >= 50000 && code < 60000:
		return "rate_limit"
	default:
		return "internal"
	}
}
