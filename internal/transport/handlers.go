package transport

import (
	"github.com/YoungGoofy/WebScanner/internal/utils/toml"
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
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
	var wg sync.WaitGroup
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
	ssEvent(c, "0", false)

	checkStatus(c, spider, &wg)

	ssEvent(c, "100", true)
}

func (s *scanner) spiderResult(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, gin.H{
		"message": "Handler not working",
	})
}

func ssEvent(c *gin.Context, progressPercentage string, completed bool) {
	c.SSEvent("progress", map[string]interface{}{
		"progressPercentage": progressPercentage,
		"completed":          completed,
	})
	c.Writer.Flush()
}

func checkStatus(c *gin.Context, spider *gozap.Spider, wg *sync.WaitGroup) {
	dataCh := make(chan gozap.UrlsInScope)
	errCh := make(chan error)
	statusCh := make(chan string)
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		spider.AsyncGetResult(dataCh, errCh, statusCh, done)
	}()
	for {
		select {
		case urls := <-dataCh:
			// TODO: add output in html, maybe js
			for _, url := range urls {
				c.SSEvent("results", map[string]string{
					"processed":          url.Processed,
					"statusReason":       url.StatusReason,
					"method":             url.Method,
					"reasonNotProcessed": url.ReasonNotProcessed,
					"messageId":          url.MessageID,
					"url":                url.URL,
					"statusCode":         url.StatusCode,
				})
				c.Writer.Flush()
			}
			//----------------------------------------------------
		case err := <-errCh:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		case status := <-statusCh:
			ssEvent(c, status, false)
			if status == "100" {
				close(done)
			}
		case <-done:
			wg.Wait()
			close(dataCh)
			close(errCh)
			close(statusCh)
			return
		}
	}
}
