package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"z42-core/internal/api/database"
)

const IdentityKey = "identity"

type IdentityData struct {
	Id    database.ObjectId
	Email string
}

func IsClientError(code int) bool {
	return code/100 == 4
}

func IsServerError(code int) bool {
	return code/100 == 5
}

func StatusFromError(c *gin.Context, err error) (*gin.Context, int, string, error) {
	switch err {
	case database.ErrInvalid:
		return c, http.StatusInternalServerError, "invalid operation", err
	case database.ErrDuplicateEntry:
		return c, http.StatusConflict, "duplicate entry", err
	case database.ErrNotFound:
		return c, http.StatusNotFound, "entry not found", err
	case database.ErrUnauthorized:
		return c, http.StatusForbidden, "authorization failed", err
	default:
		return c, http.StatusInternalServerError, "internal error", err
	}
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func ErrorResponse(c *gin.Context, code int, message string, err error) {
	if IsClientError(code) {
		zap.L().Warn(message, zap.Int("status", code), zap.Error(err))
	} else if IsServerError(code) {
		zap.L().Error(message, zap.Int("status", code), zap.Error(err))
	}
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

func ExtractUser(c *gin.Context) database.ObjectId {
	user, _ := c.Get(IdentityKey)
	return user.(*IdentityData).Id
}
