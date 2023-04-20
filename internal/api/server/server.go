package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"time"
	"z42-core/internal/api/database"
	"z42-core/internal/api/handlers"
	"z42-core/internal/api/handlers/auth"
	"z42-core/internal/api/handlers/recaptcha"
	"z42-core/internal/api/handlers/zone"
	"z42-core/internal/logger"
	"z42-core/internal/mailer"
	"z42-core/internal/upstream"
)

type Config struct {
	BindAddress        string `json:"bind_address"`
	ReadTimeout        int    `json:"read_timeout"`
	WriteTimeout       int    `json:"write_timeout"`
	MaxBodyBytes       int64  `json:"max_body_size"`
	WebServer          string `json:"web_server"`
	ApiServer          string `json:"api_server"`
	NameServer         string `json:"name_server"`
	HtmlTemplates      string `json:"html_templates"`
	RecaptchaSecretKey string `json:"recaptcha_secret_key"`
	RecaptchaServer    string `json:"recaptcha_server"`
}

func DefaultConfig() Config {
	return Config{
		BindAddress:        "localhost:8080",
		ReadTimeout:        10,
		WriteTimeout:       10,
		MaxBodyBytes:       1000000,
		WebServer:          "www.z42.com",
		ApiServer:          "api.z42.com",
		NameServer:         "ns.z42.com.",
		HtmlTemplates:      "./templates/*.tmpl",
		RecaptchaSecretKey: "RECAPTCHA_SECRET_KEY",
		RecaptchaServer:    "https://www.google.com/recaptcha/api/siteverify",
	}
}

type Server struct {
	config     *Config
	router     *gin.Engine
	httpServer *http.Server
}

func NewServer(config *Config, db *database.DataBase, mailer mailer.Mailer, u *upstream.Upstream, accessLogger *zap.Logger) *Server {
	router := gin.New()
	router.LoadHTMLGlob(config.HtmlTemplates)
	handleRecovery := func(c *gin.Context, err interface{}) {
		handlers.ErrorResponse(c, http.StatusInternalServerError, err.(string), nil)
		c.Abort()
	}
	bodySizeMiddleware := func(c *gin.Context) {
		var w http.ResponseWriter = c.Writer
		c.Request.Body = http.MaxBytesReader(w, c.Request.Body, config.MaxBodyBytes)

		c.Next()
	}
	router.Use(gin.CustomRecovery(handleRecovery))
	router.Use(bodySizeMiddleware)
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
	zoneHandler := zone.New(db, u, config.NameServer)
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
