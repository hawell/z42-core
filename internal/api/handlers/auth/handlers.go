package auth

import (
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
	"net/http"
	"time"
	"z42-core/internal/api/database"
	"z42-core/internal/api/handlers"
	"z42-core/internal/api/handlers/recaptcha"
	"z42-core/internal/mailer"
)

type storage interface {
	AddUser(u database.NewUser) (database.ObjectId, string, error)
	GetUser(name string) (database.User, error)
	Verify(code string) error
	SetRecoveryCode(userId database.ObjectId) (string, error)
	ResetPassword(code string, newPassword string) error

	AddAPIKey(userId database.ObjectId, newAPIKey database.APIKeyItem) (string, error)
	GetAPIKeys(userId database.ObjectId) ([]database.APIKeyItem, error)
	GetAPIKey(userId database.ObjectId, name string) (database.APIKeyItem, error)
	UpdateAPIKey(userId database.ObjectId, model database.APIKeyUpdate) error
	DeleteAPIKey(userId database.ObjectId, name string) error
}

type Handler struct {
	jwtMiddleWare    *jwt.GinJWTMiddleware
	db               storage
	mailer           mailer.Mailer
	serverName       string
	recaptchaHandler *recaptcha.Handler
}

const (
	emailKey   = "email"
	apiNameKey = "key_name"
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
			handlers.ErrorResponse(c, code, message, nil)
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
	group.POST("/recover", h.recaptchaHandler.MiddlewareFunc(), h.recover)
	group.PATCH("/reset", h.recaptchaHandler.MiddlewareFunc(), h.reset)
	group.POST("/login", h.recaptchaHandler.MiddlewareFunc(), h.jwtMiddleWare.LoginHandler)
	group.POST("/logout", h.jwtMiddleWare.LogoutHandler)
	group.GET("/refresh_token", h.MiddlewareFunc(), h.jwtMiddleWare.RefreshHandler)

	apikeyGroup := group.Group("/api_keys")
	apikeyGroup.Use(h.MiddlewareFunc())
	apikeyGroup.POST("", h.addAPIKey)
	apikeyGroup.GET("", h.getAPIKeys)
	apikeyGroup.GET("/:key_name", h.getAPIKey)
	apikeyGroup.PUT("/:key_name", h.updateAPIKey)
	apikeyGroup.DELETE("/:key_name", h.deleteAPIKey)
}

func (h *Handler) MiddlewareFunc() gin.HandlerFunc {
	return h.jwtMiddleWare.MiddlewareFunc()
}

func (h *Handler) signup(c *gin.Context) {
	var u NewUser
	err := c.ShouldBindBodyWith(&u, binding.JSON)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid input format", err)
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
		handlers.ErrorResponse(c, http.StatusInternalServerError, "cannot send email verification", err)
		return
	}

	handlers.SuccessResponse(c, http.StatusCreated,
		"You still have to verify your email address inorder to complete your account validation process. Please check your inbox and click the link emailed to you.",
		nil,
	)
}

func (h *Handler) verify(c *gin.Context) {
	var v verification
	err := c.ShouldBindQuery(&v)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid code", err)
		return
	}
	err = h.db.Verify(v.Code)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusInternalServerError, "verification failed", err)
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

func (h *Handler) recover(c *gin.Context) {
	var r recovery
	err := c.ShouldBindBodyWith(&r, binding.JSON)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid email", err)
		return
	}
	user, err := h.db.GetUser(r.Email)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	code, err := h.db.SetRecoveryCode(user.Id)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	err = h.mailer.SendPasswordReset(user.Email, user.Email, code)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusInternalServerError, "send password reset failed", err)
		return
	}

	handlers.SuccessResponse(c, http.StatusOK,
		"a password recovery link has been sent to your email address. Please check your inbox and click the link emailed to you.",
		nil)
}

func (h *Handler) reset(c *gin.Context) {
	var r passwordReset
	err := c.ShouldBindBodyWith(&r, binding.JSON)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid password reset request", err)
		return
	}
	err = h.db.ResetPassword(r.Code, r.Password)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}

	handlers.SuccessResponse(c, http.StatusAccepted,
		"your password has been updated successfully. you may now login using your new password",
		nil,
	)
}

func (h *Handler) addAPIKey(c *gin.Context) {
	userId := handlers.ExtractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing", nil)
		return
	}

	var r NewAPIKeyRequest
	err := c.ShouldBindJSON(&r)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "binding request failed", err)
		return
	}

	model := database.APIKeyItem{
		Name:     r.Name,
		Scope:    r.Scope,
		ZoneName: r.ZoneName,
		Enabled:  r.Enabled,
	}
	key, err := h.db.AddAPIKey(userId, model)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	resp := NewAPIKeyResponse{
		Name:     r.Name,
		Key:      key,
		ZoneName: r.ZoneName,
		Scope:    r.Scope,
		Enabled:  r.Enabled,
	}
	handlers.SuccessResponse(c, http.StatusCreated, "successful", resp)
}

func (h *Handler) getAPIKeys(c *gin.Context) {
	userId := handlers.ExtractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing", nil)
		return
	}
	items, err := h.db.GetAPIKeys(userId)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusOK, "successful", items)
}

func (h *Handler) getAPIKey(c *gin.Context) {
	userId := handlers.ExtractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing", nil)
		return
	}
	name := c.Param(apiNameKey)
	if name == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "api name missing", nil)
		return
	}

	item, err := h.db.GetAPIKey(userId, name)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessResponse(c, http.StatusOK, "successful", item)
}

func (h *Handler) updateAPIKey(c *gin.Context) {
	userId := handlers.ExtractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing", nil)
		return
	}
	name := c.Param(apiNameKey)
	if name == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "api name missing", nil)
		return
	}

	var r UpdateAPIKeyRequest
	err := c.ShouldBindJSON(&r)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "binding request failed", err)
		return
	}

	model := database.APIKeyUpdate{
		Name:    name,
		Scope:   r.Scope,
		Enabled: r.Enabled,
	}
	err = h.db.UpdateAPIKey(userId, model)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusOK, "successful", name)
}

func (h *Handler) deleteAPIKey(c *gin.Context) {
	userId := handlers.ExtractUser(c)
	if userId == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "user missing", nil)
		return
	}
	name := c.Param(apiNameKey)
	if name == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "api name missing", nil)
		return
	}

	err := h.db.DeleteAPIKey(userId, name)
	if err != nil {
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	handlers.SuccessfulOperationResponse(c, http.StatusOK, "successful", name)

}
