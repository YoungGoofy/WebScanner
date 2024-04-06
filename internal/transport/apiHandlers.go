package transport

import (
	"github.com/YoungGoofy/WebScanner/internal/utils/toml"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func postAPIKey(c *gin.Context) {
	var newApiKey struct {
		ApiKey string `form:"apiKey" binding:"required"`
	}
	if err := c.ShouldBind(&newApiKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := toml.PostApiKeyToToml(newApiKey.ApiKey); err != nil {
		log.Println(err)
		return
	}
	c.HTML(http.StatusOK, "index.html", gin.H{"apiKey": newApiKey.ApiKey})
}
