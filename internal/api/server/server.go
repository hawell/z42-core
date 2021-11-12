package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"github.com/hawell/z42/internal/api/handlers/auth"
	"github.com/hawell/z42/internal/api/handlers/recaptcha"
	"github.com/hawell/z42/internal/api/handlers/zone"
	"github.com/hawell/z42/internal/logger"
	"github.com/hawell/z42/internal/mailer"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Config struct {
	BindAddress        string `json:"bind_address,default:localhost:8080"`
	ReadTimeout        int    `json:"read_timeout,default:10"`
	WriteTimeout       int    `json:"write_timeout,default:10"`
	WebServer          string `json:"web_server"`
	ApiServer          string `json:"api_server"`
	NameServer         string `json:"name_server"`
	HtmlTemplates      string `json:"html_templates"`
	RecaptchaSecretKey string `json:"recaptcha_secret_key"`
	RecaptchaServer    string `json:"recaptcha_server"`
}

type Server struct {
	config     *Config
	router     *gin.Engine
	httpServer *http.Server
}

func NewServer(config *Config, db *database.DataBase, mailer mailer.Mailer, accessLogger *zap.Logger) *Server {
	router := gin.New()
	router.LoadHTMLGlob(config.HtmlTemplates)
	handleRecovery := func(c *gin.Context, err interface{}) {
		handlers.ErrorResponse(c, http.StatusInternalServerError, err.(string))
		c.Abort()
	}
	router.Use(gin.CustomRecovery(handleRecovery))
	router.Use(logger.MiddlewareFunc(accessLogger))
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, ResponseType, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	s := &http.Server{
		Addr:           config.BindAddress,
		Handler:        router,
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	authGroup := router.Group("/auth")
	recaptchaHandler := recaptcha.New(config.RecaptchaServer, config.RecaptchaSecretKey)
	authHandler := auth.New(db, mailer, recaptchaHandler, config.WebServer)
	authHandler.RegisterHandlers(authGroup)

	zoneGroup := router.Group("/zones")
	zoneGroup.Use(authHandler.MiddlewareFunc())
	zoneHandler := zone.New(db, config.NameServer)
	zoneHandler.RegisterHandlers(zoneGroup)

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
