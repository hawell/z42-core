package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"net/http"
)

const IdentityKey = "identity"

type IdentityData struct {
	Id    database.ObjectId
	Email string
}

func StatusFromError(c *gin.Context, err error) (*gin.Context, int, string) {
	switch err {
	case database.ErrInvalid:
		return c, http.StatusForbidden, "invalid request"
	case database.ErrDuplicateEntry:
		return c, http.StatusConflict, "duplicate entry"
	case database.ErrNotFound:
		return c, http.StatusNotFound, "entry not found"
	case database.ErrUnauthorized:
		return c, http.StatusUnauthorized, "authorization failed"
	default:
		return c, http.StatusInternalServerError, "internal error"
	}
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
	})
}

func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func SuccessfulOperationResponse(c *gin.Context, code int, message string, name string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data: map[string]string{
			"name": name,
		},
	})
}
