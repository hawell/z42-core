package auth

import (
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"go.uber.org/zap"
	"time"
)

type storage interface {
	GetUser(name string) (database.User, error)
}

type Handler struct {
	jwtMiddleWare *jwt.GinJWTMiddleware
	db storage
}

type loginCredentials struct {
	Email string `form:"email" json:"email" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

const (
	emailKey = "email"
)

func New(db storage) *Handler {
	handler := Handler{
		db: db,
	}
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "z42 zone",
		Key:         []byte("secret key"),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: handlers.IdentityKey,
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals loginCredentials
			if err := c.ShouldBind(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			email := loginVals.Email
			password := loginVals.Password

			user, err := handler.db.GetUser(email)
			if err != nil {
				zap.L().Error("user not found")
				return nil, jwt.ErrFailedAuthentication
			}

			if !database.CheckPasswordHash(password, user.Password) {
				zap.L().Error("password mismatch")
				return nil, jwt.ErrFailedAuthentication
			}
			return &user, nil
		},
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*database.User); ok {
				return jwt.MapClaims{
					handlers.IdentityKey: v.Id,
					emailKey:    v.Email,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &database.User{
				Id: int64(claims[handlers.IdentityKey].(float64)),
				Email: claims[emailKey].(string),
			}
		},
		TokenLookup: "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		SendCookie: true,
		TimeFunc: time.Now,
	})

	if err != nil {
		zap.L().Fatal("jwt error", zap.Error(err))
	}

	return &Handler{jwtMiddleWare: jwtMiddleware}
}

func (h *Handler) RegisterHandlers(group *gin.RouterGroup) {
	group.POST("/login", h.jwtMiddleWare.LoginHandler)
	group.POST("/logout", h.jwtMiddleWare.LogoutHandler)
	group.GET("/refresh_token", h.MiddlewareFunc(), h.jwtMiddleWare.RefreshHandler)
}

func (h *Handler) MiddlewareFunc() gin.HandlerFunc {
	return h.jwtMiddleWare.MiddlewareFunc()
}

