package server

import (
	"context"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Config struct {
	BindAddress string `json:"bind_address,default:localhost:8080"`
	ReadTimeout int `json:"read_timeout,default:10"`
	WriteTimeout int `json:"write_timeout,default:10"`
}

type Server struct {
	config *Config
	router *gin.Engine
	httpServer *http.Server
}

func dbMiddleware(db *database.DataBase) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("database", db)
		c.Next()
	}
}

type loginCredentials struct {
	Email string `form:"email" json:"email" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

var (
	identityKey = "id"
	emailKey = "email"
)
func NewServer(config *Config, db *database.DataBase) *Server {
	router := gin.Default()

	s := &http.Server{
		Addr:           config.BindAddress,
		Handler:        router,
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	router.Use(dbMiddleware(db))

	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "z42 zone",
		Key:         []byte("secret key"),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: identityKey,
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals loginCredentials
			if err := c.ShouldBind(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			email := loginVals.Email
			password := loginVals.Password

			db := extractDataBase(c)
			if db == nil {
				zap.L().Error("no db")
				return nil, jwt.ErrFailedAuthentication
			}

			user, err := db.GetUser(email)
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
					identityKey: v.Id,
					emailKey:    v.Email,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &database.User{
				Id: int64(claims[identityKey].(float64)),
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

	router.POST("/login", authMiddleware.LoginHandler)
	router.POST("/logout", authMiddleware.LogoutHandler)
	router.GET("/refresh_token", authMiddleware.MiddlewareFunc(), authMiddleware.RefreshHandler)

	zones := router.Group("/zones")
	{
		zones.Use(authMiddleware.MiddlewareFunc())

		zones.GET("", GetZones)
		zones.POST("", AddZone)

		zone := zones.Group("/:zone")
		{
			zone.GET("", GetZone)
			zone.PUT("", UpdateZone)
			zone.DELETE("", DeleteZone)

			locations := zone.Group("/locations")
			{
				locations.GET("", GetLocations)
				locations.POST("", AddLocation)

				location := locations.Group("/:location")
				{
					location.GET("", GetLocation)
					location.PUT("", UpdateLocation)
					location.DELETE("", DeleteLocation)

					rrsets := location.Group("/rrsets")
					{
						rrsets.GET("", GetRecordSets)
						rrsets.POST("", AddRecordSet)

						rrset := rrsets.Group("/:rtype")
						{
							rrset.GET("", GetRecordSet)
							rrset.PUT("", UpdateRecordSet)
							rrset.DELETE("", DeleteRecordSet)
						}
					}
				}
			}
		}
	}

	return &Server{
		config:     config,
		router:     router,
		httpServer: s,
	}
}

func (s *Server) ListenAndServer() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	return s.httpServer.Shutdown(context.Background())
}
