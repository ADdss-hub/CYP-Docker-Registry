// Package common provides shared utilities for the container registry.
package common

// ErrorCode represents a standardized error code.
type ErrorCode string

// Error codes
const (
	ErrImageNotFound   ErrorCode = "IMAGE_NOT_FOUND"
	ErrBlobNotFound    ErrorCode = "BLOB_NOT_FOUND"
	ErrInvalidManifest ErrorCode = "INVALID_MANIFEST"
	ErrStorageFull     ErrorCode = "STORAGE_FULL"
	ErrUpstreamError   ErrorCode = "UPSTREAM_ERROR"
	ErrAuthFailed      ErrorCode = "AUTH_FAILED"
	ErrInternalError   ErrorCode = "INTERNAL_ERROR"
	ErrInvalidRequest  ErrorCode = "INVALID_REQUEST"
	ErrNotFound        ErrorCode = "NOT_FOUND"
)

// HTTPStatus returns the HTTP status code for the error code.
func (e ErrorCode) HTTPStatus() int {
	switch e {
	case ErrImageNotFound, ErrBlobNotFound, ErrNotFound:
		return 404
	case ErrInvalidManifest, ErrInvalidRequest:
		return 400
	case ErrStorageFull:
		return 507
	case ErrUpstreamError:
		return 502
	case ErrAuthFailed:
		return 401
	default:
		return 500
	}
}

// Message returns the default message for the error code.
func (e ErrorCode) Message() string {
	switch e {
	case ErrImageNotFound:
		return "镜像不存在"
	case ErrBlobNotFound:
		return "镜像层不存在"
	case ErrInvalidManifest:
		return "无效的镜像清单"
	case ErrStorageFull:
		return "存储空间不足"
	case ErrUpstreamError:
		return "上游仓库错误"
	case ErrAuthFailed:
		return "认证失败"
	case ErrInvalidRequest:
		return "无效的请求"
	case ErrNotFound:
		return "资源不存在"
	default:
		return "内部错误"
	}
}
