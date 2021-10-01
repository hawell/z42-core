package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers/auth"
	"github.com/hawell/z42/internal/api/handlers/zone"
	"github.com/hawell/z42/internal/mailer"
	"net/http"
	"time"
)

type Config struct {
	BindAddress  string `json:"bind_address,default:localhost:8080"`
	ReadTimeout  int    `json:"read_timeout,default:10"`
	WriteTimeout int    `json:"write_timeout,default:10"`
	AuthoritativeServer string `json:"authoritative_server"`
}

type Server struct {
	config     *Config
	router     *gin.Engine
	httpServer *http.Server
}

func NewServer(config *Config, db *database.DataBase, mailer mailer.Mailer) *Server {
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, ResponseType, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

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
	authHandler := auth.New(db, mailer)
	authHandler.RegisterHandlers(authGroup)

	zoneGroup := router.Group("/zones")
	zoneGroup.Use(authHandler.MiddlewareFunc())
	zoneHandler := zone.New(db, config.AuthoritativeServer)
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
