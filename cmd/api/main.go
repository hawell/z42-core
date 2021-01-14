package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hawell/z42/internal/api"
	"github.com/hawell/z42/internal/api/database"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"time"
)

func DBMiddleware(db *database.DataBase) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("database", db)
		c.Next()
	}
}

func main() {
	configPtr := flag.String("c", "config.json", "path to config file")
	flag.Parse()
	configFile := *configPtr
	config, err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	eventLoggerConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.ErrorLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	eventLogger, err := eventLoggerConfig.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(eventLogger)

	db, err := database.Connect(config.DBConnectionString)
	if err != nil {
		panic(err)
	}

	router := gin.Default()

	s := &http.Server{
		Addr:           config.BindAddress,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	router.Use(DBMiddleware(db))

	zones := router.Group("/zones")
	{
		zones.GET("", api.GetZones)
		zones.POST("", api.AddZone)

		zone := zones.Group("/:zone")
		{
			zone.GET("", api.GetZone)
			zone.PUT("", api.UpdateZone)
			zone.DELETE("", api.DeleteZone)

			locations := zone.Group("/locations")
			{
				locations.GET("", api.GetLocations)
				locations.POST("", api.AddLocation)

				location := locations.Group("/:location")
				{
					location.GET("", api.GetLocation)
					location.PUT("", api.UpdateLocation)
					location.DELETE("", api.DeleteLocation)

					rrsets := location.Group("/rrsets")
					{
						rrsets.GET("", api.GetRecordSets)
						rrsets.POST("", api.AddRecordSet)

						rrset := rrsets.Group("/:rtype")
						{
							rrset.GET("", api.GetRecordSet)
							rrset.PUT("", api.UpdateRecordSet)
							rrset.DELETE("", api.DeleteRecordSet)
						}
					}
				}
			}
		}
	}

	err = s.ListenAndServe()
	fmt.Println(err)
}
