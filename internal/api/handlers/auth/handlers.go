package auth

import (
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"github.com/hawell/z42/internal/api/handlers/recaptcha"
	"github.com/hawell/z42/internal/mailer"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type storage interface {
	AddUser(u database.NewUser) (database.ObjectId, string, error)
	GetUser(name string) (database.User, error)
	Verify(code string) error
}

type Handler struct {
	jwtMiddleWare    *jwt.GinJWTMiddleware
	db               storage
	mailer           mailer.Mailer
	serverName       string
	recaptchaHandler *recaptcha.Handler
}

const (
	emailKey = "email"
)

func New(db storage, mailer mailer.Mailer, recaptchaHandler *recaptcha.Handler, serverName string) *Handler {
	handler := &Handler{
		db:               db,
		mailer:           mailer,
		serverName:       serverName,
		recaptchaHandler: recaptchaHandler,
	}
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "z42 zone",
		Key:         []byte("secret key"),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: handlers.IdentityKey,
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginValues loginCredentials
			if err := c.ShouldBindBodyWith(&loginValues, binding.JSON); err != nil {
				zap.L().Warn("missing login values")
				return "", jwt.ErrMissingLoginValues
			}
			email := loginValues.Email
			password := loginValues.Password

			user, err := handler.db.GetUser(email)
			if err != nil {
				zap.L().Warn("user not found")
				return nil, jwt.ErrFailedAuthentication
			}

			if user.Status != database.UserStatusActive {
				zap.L().Warn("user not active")
				return nil, jwt.ErrFailedAuthentication
			}

			if !database.CheckPasswordHash(password, user.Password) {
				zap.L().Warn("password mismatch")
				return nil, jwt.ErrFailedAuthentication
			}
			return &handlers.IdentityData{Id: user.Id, Email: user.Email}, nil
		},
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*handlers.IdentityData); ok {
				return jwt.MapClaims{
					handlers.IdentityKey: v.Id,
					emailKey:             v.Email,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &handlers.IdentityData{
				Id:    database.ObjectId(claims[handlers.IdentityKey].(string)),
				Email: claims[emailKey].(string),
			}
		},
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			c.JSON(http.StatusOK, &authenticationToken{
				Code:   http.StatusOK,
				Token:  token,
				Expire: expire.Format(time.RFC3339),
			})
		},
		LogoutResponse: func(c *gin.Context, code int) {
			handlers.SuccessResponse(c, code, "logout successful", nil)
		},
		RefreshResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			c.JSON(http.StatusOK, &authenticationToken{
				Code:   http.StatusOK,
				Token:  token,
				Expire: expire.Format(time.RFC3339),
			})
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			handlers.ErrorResponse(c, code, message)
		},
		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		SendCookie:    true,
		TimeFunc:      time.Now,
	})

	if err != nil {
		zap.L().Fatal("jwt error", zap.Error(err))
	}
	handler.jwtMiddleWare = jwtMiddleware
	return handler
}

func (h *Handler) RegisterHandlers(group *gin.RouterGroup) {
	group.POST("/signup", h.recaptchaHandler.MiddlewareFunc(), h.signup)
	group.POST("/verify", h.verify)
	group.POST("/login", h.recaptchaHandler.MiddlewareFunc(), h.jwtMiddleWare.LoginHandler)
	group.POST("/logout", h.jwtMiddleWare.LogoutHandler)
	group.GET("/refresh_token", h.MiddlewareFunc(), h.jwtMiddleWare.RefreshHandler)
}

func (h *Handler) MiddlewareFunc() gin.HandlerFunc {
	return h.jwtMiddleWare.MiddlewareFunc()
}

func (h *Handler) signup(c *gin.Context) {
	var u NewUser
	err := c.ShouldBindBodyWith(&u, binding.JSON)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid input format")
		return
	}
	model := database.NewUser{
		Email:    u.Email,
		Password: u.Password,
		Status:   database.UserStatusPending,
	}
	_, code, err := h.db.AddUser(model)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	err = h.mailer.SendEMailVerification(u.Email, u.Email, code)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	handlers.SuccessResponse(c, http.StatusCreated, "successful", nil)
}

func (h *Handler) verify(c *gin.Context) {
	var v verification
	err := c.ShouldBindQuery(&v)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid code")
		return
	}
	err = h.db.Verify(v.Code)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(
		http.StatusOK,
		"verification-successful.tmpl",
		gin.H{
			"Server": h.serverName,
		},
	)
}
