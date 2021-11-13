package recaptcha

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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

type token struct {
	Value string `form:"recaptcha_token" json:"recaptcha_token" binding:"required"`
}

func New(server string, secretKey string) *Handler {
	return &Handler{
		client: &http.Client{
			Timeout: time.Duration(10) * time.Second,
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
	var t token
	err := ctx.ShouldBindBodyWith(&t, binding.JSON)
	if err != nil || t.Value == "" {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, "recaptcha token is missing", err)
		ctx.Abort()
		return
	}

	resp, err := h.client.PostForm(h.server,
		url.Values{"secret": {h.secretKey}, "response": {t.Value}})
	if err != nil {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, "recaptcha PostForm failed", err)
		ctx.Abort()
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, "reading response body failed", err)
		ctx.Abort()
		return
	}

	var responseData Response
	if err := json.Unmarshal(body, &responseData); err != nil {
		handlers.ErrorResponse(ctx, http.StatusBadRequest, "unmarshal response body failed", err)
		ctx.Abort()
		return
	}

	if responseData.Success == false || responseData.Action != "login" {
		handlers.ErrorResponse(ctx, http.StatusForbidden, "recaptcha validation failed", nil)
		ctx.Abort()
		return
	}

	ctx.Next()
}
