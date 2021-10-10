package recaptcha

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/handlers"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type Handler struct {
	client    *http.Client
	secretKey string
	server    string
}

const TokenKey = "recaptcha_token"

func New(server string, secretKey string) *Handler {
	return &Handler{
		client: &http.Client{
			Timeout: time.Duration(5) * time.Second,
		},
		secretKey: secretKey,
		server:    server,
	}
}

func (h *Handler) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h.VerifyReCaptcha(ctx)
	}
}

func (h *Handler) VerifyReCaptcha(ctx *gin.Context) {
	token := ctx.Query(TokenKey)
	if token == "" {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, "recaptcha token is missing")
		ctx.Abort()
		return
	}

	resp, err := h.client.PostForm(h.server,
		url.Values{"secret": {h.secretKey}, "response": {token}})
	if err != nil {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, err.Error())
		ctx.Abort()
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, err.Error())
		ctx.Abort()
		return
	}

	var responseData Response
	if err := json.Unmarshal(body, &responseData); err != nil {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, err.Error())
		ctx.Abort()
		return
	}

	if responseData.Success == false || responseData.Action != "login" {
		handlers.ErrorResponse(ctx, http.StatusForbidden, "recaptcha validation failed")
		ctx.Abort()
		return
	}

	ctx.Next()
}
