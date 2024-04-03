package transport

import (
	"github.com/YoungGoofy/WebScanner/internal/utils/toml"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func MainHandler(api *gin.Engine) {
	{
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
	}
	api.GET("/scan", startScan)
	api.GET("/scan/spider", spider)
	api.POST("/key", postAPIKey)
}

func postAPIKey(c *gin.Context) {
	var newApiKey struct {
		ApiKey string `json:"apiKey"`
	}
	c.Header("Content-Type", "application/json")
	if err := c.BindJSON(&newApiKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err := toml.PostApiKeyToToml(newApiKey.ApiKey); err != nil {
		log.Println(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Add new api-key",
	})
}

func startScan(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{
		"message": "Handler not working",
	})
}

func spider(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{
		"message": "Handler not working",
	})
}
