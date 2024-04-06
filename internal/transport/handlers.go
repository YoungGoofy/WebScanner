package transport

import (
	"github.com/YoungGoofy/WebScanner/internal/utils/toml"
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/gin-gonic/gin"
	"net/http"
)

type scanner struct {
	gozap.Scan
}

func newScanner(apiKey string) *scanner {
	newScan := gozap.NewScan("", apiKey)
	//newSpider := gozap.NewSpider(*newScan)
	return &scanner{*newScan}
}

func MainHandler(api *gin.Engine) {
	key := toml.GetApiKeyFromToml()
	scan := newScanner(key)
	api.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Home Page",
		})
	})
	api.GET("/scan", scan.startScan)
	api.GET("/scan/spiderResult", scan.spiderResult)
	api.GET("/settings", func(c *gin.Context) {
		c.HTML(http.StatusOK, "settings.html", gin.H{
			"title": "Settings Page",
			"key":   key,
		})
	})
	api.POST("/settings/addKey", postAPIKey)
}

func (s *scanner) startScan(c *gin.Context) {
	newUrl := struct {
		Url string `form:"url" binding:"required"`
	}{}
	if err := c.ShouldBind(&newUrl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//if err := s.GetSessionId(); err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{
	//		"error": err,
	//	})
	//}
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{
		"message": newUrl.Url,
	})
}

func (s *scanner) spiderResult(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{
		"message": "Handler not working",
	})
}
