package handlers

import (
    "github.com/hawell/z42/internal/api/database"
    "net/http"
)

const IdentityKey = "identity"

func StatusFromError(err error) (int, string) {
    switch err {
    case database.ErrInvalid:
        return http.StatusForbidden, "invalid request"
    case database.ErrDuplicateEntry:
        return http.StatusConflict, "duplicate entry"
    case database.ErrNotFound:
        return http.StatusNotFound, "entry not found"
    default:
        return http.StatusInternalServerError, "internal error"
    }
}