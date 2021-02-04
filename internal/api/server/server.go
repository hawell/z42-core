package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api/database"
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

	zones := router.Group("/zones")
	{
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
