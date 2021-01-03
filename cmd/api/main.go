package main

import "github.com/gin-gonic/gin"

var DefaultHandler = func(c *gin.Context) {
		c.String(200, c.Request.URL.String())
}

func main() {
	router := gin.Default()


	zones := router.Group("/zones")
	{
		zones.GET("", DefaultHandler)
		zones.POST("", DefaultHandler)

		zone := zones.Group("/:zone")
		{
			zone.GET("", DefaultHandler)
			zone.PUT("", DefaultHandler)
			zone.DELETE("", DefaultHandler)

			locations := zone.Group("/locations")
			{
				locations.GET("", DefaultHandler)
				locations.POST("", DefaultHandler)

				location := locations.Group("/:location")
				{
					location.GET("", DefaultHandler)
					location.PUT("", DefaultHandler)
					location.DELETE("", DefaultHandler)

					rrsets := location.Group("/rrsets")
					{
						rrsets.GET("", DefaultHandler)
						rrsets.POST("", DefaultHandler)

						rrset := rrsets.Group("/:rtype")
						{
							rrset.GET("", DefaultHandler)
							rrset.PUT("", DefaultHandler)
							rrset.DELETE("", DefaultHandler)
						}
					}
				}
			}
		}
	}
	router.Run()
}
