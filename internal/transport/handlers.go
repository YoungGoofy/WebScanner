package transport

import (
	"github.com/YoungGoofy/WebScanner/internal/utils/toml"
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type scanner struct {
	gozap.MainScan
}

func newScanner(apiKey string) *scanner {
	newScan := gozap.NewMainScan("", apiKey)
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
		Url string `form:"url"`
	}{}
	if err := c.ShouldBind(&newUrl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.AddUrl(newUrl.Url)
	spider := gozap.NewSpider(s.MainScan)
	if err := spider.GetSessionId(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Отправляем начальное состояние сканирования
	c.SSEvent("progress", map[string]interface{}{
		"progressPercentage": 0,
		"completed":          false,
	})
	c.Writer.Flush()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		status, err := spider.GetStatus()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if status == "100" {
			break
		}
		c.SSEvent("progress", map[string]interface{}{
			"progressPercentage": status,
			"completed":          false,
		})
		c.Writer.Flush()
	}

	c.SSEvent("progress", map[string]interface{}{
		"progressPercentage": 100,
		"completed":          true,
	})
	c.Writer.Flush()
}

func (s *scanner) spiderResult(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{
		"message": "Handler not working",
	})
}
