package main

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/api/drivers/nearby", func(c *gin.Context) {
		lat := c.Query("lat")
		lng := c.Query("lng")
		radius := c.Query("radius")

		// Build JSON payload for geo service, including radius
		payload := []byte(`{"rider_location":{"lat":` + lat + `,"lng":` + lng + `},"destination":{"lat":` + lat + `,"lng":` + lng + `},"ride_type":"standard","radius":` + radius + `}`)

		geoURL := "http://localhost:8083/api/v1/geo/nearby-drivers"
		req, err := http.NewRequest("POST", geoURL, ioutil.NopCloser(bytes.NewReader(payload)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request to geo service"})
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Geo service unavailable"})
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read geo service response"})
			return
		}

		c.Data(resp.StatusCode, "application/json", body)
	})

	r.Run(":8080")
}
