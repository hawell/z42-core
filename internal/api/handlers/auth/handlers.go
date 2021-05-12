package auth

import (
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"github.com/hawell/z42/pkg/hiredis"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type storage interface {
	AddUser(u database.User) (int64, error)
	GetUser(name string) (database.User, error)
	AddVerification(user string, verificationType string) (string, error)
	Verify(code string) error
}

type Handler struct {
	jwtMiddleWare *jwt.GinJWTMiddleware
	db storage
	redis *hiredis.Redis
}

type loginCredentials struct {
	Email string `form:"email" json:"email" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

const (
	emailKey = "email"
)

func New(db storage, redis *hiredis.Redis) *Handler {
	handler := &Handler{
		db: db,
		redis: redis,
	}
	jwtMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "z42 zone",
		Key:         []byte("secret key"),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: handlers.IdentityKey,
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginValues loginCredentials
			if err := c.ShouldBind(&loginValues); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			email := loginValues.Email
			password := loginValues.Password

			user, err := handler.db.GetUser(email)
			if err != nil {
				zap.L().Error("user not found")
				return nil, jwt.ErrFailedAuthentication
			}

			if user.Status != database.UserStatusActive {
				zap.L().Error("user not active")
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
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			c.JSON(http.StatusOK, gin.H{
				"code":   http.StatusOK,
				"token":  token,
				"expire": expire.Format(time.RFC3339),
			})
		},
		LogoutResponse: func(c *gin.Context, code int) {
			handlers.SuccessResponse(c, code, "logout successful")
		},
		RefreshResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			c.JSON(http.StatusOK, gin.H{
				"code":   http.StatusOK,
				"token":  token,
				"expire": expire.Format(time.RFC3339),
			})
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			handlers.ErrorResponse(c, code, message)
		},
		TokenLookup: "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		SendCookie: true,
		TimeFunc: time.Now,
	})

	if err != nil {
		zap.L().Fatal("jwt error", zap.Error(err))
	}
	handler.jwtMiddleWare = jwtMiddleware
	return handler
}

func (h *Handler) RegisterHandlers(group *gin.RouterGroup) {
	group.POST("/signup", h.signup)
	group.POST("/verify", h.verify)
	group.POST("/login", h.jwtMiddleWare.LoginHandler)
	group.POST("/logout", h.jwtMiddleWare.LogoutHandler)
	group.GET("/refresh_token", h.MiddlewareFunc(), h.jwtMiddleWare.RefreshHandler)
}

func (h *Handler) MiddlewareFunc() gin.HandlerFunc {
	return h.jwtMiddleWare.MiddlewareFunc()
}

func (h *Handler) signup(c *gin.Context) {
	var u database.User
	err := c.ShouldBindJSON(&u)
	if err != nil {
		handlers.ErrorResponse(c, http.StatusBadRequest, "invalid input format")
		return
	}
	u.Status = database.UserStatusPending
	_, err = h.db.AddUser(u)
	if err != nil {
		zap.L().Error("DataBase.addUser()", zap.Error(err))
		handlers.ErrorResponse(handlers.StatusFromError(c, err))
		return
	}
	code, err := h.db.AddVerification(u.Email, database.VerificationTypeSignup)
	if err != nil {
		zap.L().Error("add verification code failed", zap.Error(err))
		handlers.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	// TODO : refactor to function
	_, err = h.redis.XAdd("email_verification", hiredis.StreamItem{Key: u.Email, Value: code})
	if err != nil {
		zap.L().Error("send verification code failed", zap.Error(err))
		handlers.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	handlers.SuccessResponse(c, http.StatusCreated, "successful")
}

func (h *Handler) verify(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		handlers.ErrorResponse(c, http.StatusBadRequest, "code missing")
		return
	}
	err := h.db.Verify(code)
	if err != nil {
		zap.L().Error("verification failed", zap.Error(err))
		handlers.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	handlers.SuccessResponse(c, http.StatusNoContent, "successful")
}