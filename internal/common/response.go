package common

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error details in API response.
type ErrorInfo struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// SuccessResponse sends a successful JSON response.
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Success: true,
		Data:    data,
	})
}

// ErrorResponse sends an error JSON response.
func ErrorResponse(c *gin.Context, code ErrorCode, details map[string]interface{}) {
	c.JSON(code.HTTPStatus(), Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      code,
			Message:   code.Message(),
			Details:   details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ErrorResponseWithMessage sends an error JSON response with custom message.
func ErrorResponseWithMessage(c *gin.Context, code ErrorCode, message string, details map[string]interface{}) {
	c.JSON(code.HTTPStatus(), Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      code,
			Message:   message,
			Details:   details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// NewErrorInfo creates a new ErrorInfo instance.
func NewErrorInfo(code ErrorCode, details map[string]interface{}) *ErrorInfo {
	return &ErrorInfo{
		Code:      code,
		Message:   code.Message(),
		Details:   details,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
