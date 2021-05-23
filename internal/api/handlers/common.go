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

func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
	})
}

func SuccessResponse(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
	})
}
